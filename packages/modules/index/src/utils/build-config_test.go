package utils_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
	"github.com/hitkey-io/medusago/packages/modules/index/src/utils"
)

// shape renders a representation like the upstream dump
// (integration-tests/http-diff/testapp/dump-index-schema.js).
func shape(s *utils.SchemaObjectRepresentation) any {
	entities := []any{}
	for _, key := range s.Keys {
		rep := s.Entities[key]
		var moduleConfig any
		if rep.ModuleConfig.ServiceName != "" {
			rels := []any{}
			for _, r := range rep.ModuleConfig.Relationships {
				rels = append(rels, map[string]any{"serviceName": r.ServiceName, "foreignKey": r.ForeignKey})
			}
			pks := []any{}
			for _, pk := range rep.ModuleConfig.PrimaryKeys {
				pks = append(pks, pk)
			}
			moduleConfig = map[string]any{"serviceName": rep.ModuleConfig.ServiceName, "isLink": rep.ModuleConfig.IsLink, "primaryKeys": pks, "relationships": rels}
		}
		parents := []any{}
		for _, p := range rep.Parents {
			var inSchemaRef any
			if p.InSchemaRef != nil {
				inSchemaRef = p.InSchemaRef.Entity
			}
			var isList any = p.IsList
			if p.IsListUnset {
				isList = nil
			}
			parents = append(parents, map[string]any{"ref": p.Ref.Entity, "inSchemaRef": inSchemaRef, "targetProp": p.TargetProp, "inverseSideProp": p.InverseSideProp, "isList": isList, "isInverse": p.IsInverse})
		}
		entities = append(entities, map[string]any{
			"key": key, "entity": rep.Entity, "alias": rep.Alias, "fields": strs(rep.Fields), "listeners": strs(rep.Listeners),
			"moduleConfig": moduleConfig, "parents": parents,
		})
	}
	props := []any{}
	for _, path := range s.PropertiesKeys {
		e := s.PropertiesMap[path]
		props = append(props, map[string]any{"path": path, "ref": e.Ref.Entity, "shortCutOf": e.ShortCutOf, "isInverse": e.IsInverse, "isList": e.IsList})
	}
	linkable := map[string]any{}
	for k, v := range s.LinkableKeys {
		m := map[string]any{}
		for a, b := range v {
			m[a] = b
		}
		linkable[k] = m
	}
	types := map[string]any{}
	for k, v := range s.Types {
		m := map[string]any{}
		for a, b := range v {
			m[a] = b
		}
		types[k] = m
	}
	return map[string]any{"entities": entities, "propertiesMap": props, "linkableKeys": linkable, "types": types}
}

func strs(in []string) []any {
	out := []any{}
	for _, s := range in {
		out = append(out, s)
	}
	return out
}

// normalize maps upstream's undefined (null in the dump) to the Go zero
// values: optional strings and the booleans that are only read as truthy.
func normalize(v any, key string) any {
	switch x := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, val := range x {
			out[k] = normalize(val, k)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, val := range x {
			out[i] = normalize(val, key)
		}
		return out
	case nil:
		switch key {
		case "alias", "inverseSideProp", "shortCutOf":
			return ""
		case "isInverse":
			return false
		}
	}
	return v
}

// The propertiesMap isList of the dump is undefined (null) where Go has
// false; parents keep null (an undefined isList matters for shortcuts).
func normalizeProps(v any) {
	m := v.(map[string]any)
	for _, p := range m["propertiesMap"].([]any) {
		e := p.(map[string]any)
		if e["isList"] == nil {
			e["isList"] = false
		}
	}
}

// BuildSchemaObjectRepresentation over the joiner configs of the default
// modules and links must give what upstream builds for the same schema
// (testdata/index-schemas: the default schema and the .graphql files,
// dumped from the reference application).
func TestBuildSchemaObjectRepresentationMatchesUpstream(t *testing.T) {
	qc, err := loaders.BuildQueryConfig(config.DefineConfig(config.InputConfig{}))
	if err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob("testdata/index-schemas/*.json")
	if len(files) == 0 {
		t.Fatal("no vectors")
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".json")
		t.Run(name, func(t *testing.T) {
			schema := utils.DefaultSchema
			if name != "default" {
				data, err := os.ReadFile(strings.TrimSuffix(file, ".json") + ".graphql")
				if err != nil {
					t.Fatal(err)
				}
				schema = string(data)
			}
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var want any
			if err := json.Unmarshal(raw, &want); err != nil {
				t.Fatal(err)
			}
			got, err := utils.BuildSchemaObjectRepresentation(utils.BaseGraphqlSchema+schema, qc.JoinerConfigs, logger)
			if wantErr, ok := want.(map[string]any)["error"].(string); ok {
				if err == nil || err.Error() != wantErr {
					t.Fatalf("error: want %q, got %v", wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			// round trip through JSON so that both sides have the same types
			gotJSON, _ := json.Marshal(shape(got))
			var gotAny any
			_ = json.Unmarshal(gotJSON, &gotAny)
			w, g := normalize(want, ""), normalize(gotAny, "")
			normalizeProps(w)
			normalizeProps(g)
			for _, k := range []string{"entities", "propertiesMap", "linkableKeys", "types"} {
				wv, gv := w.(map[string]any)[k], g.(map[string]any)[k]
				if !reflect.DeepEqual(wv, gv) {
					wj, _ := json.MarshalIndent(wv, "", " ")
					gj, _ := json.MarshalIndent(gv, "", " ")
					t.Errorf("%s differs\nwant %s\ngot  %s", k, wj, gj)
				}
			}
		})
	}
}
