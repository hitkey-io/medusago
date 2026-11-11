package brand

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
)

// testdata/upstream.json is written by testapp/dump-brand.js from the
// JavaScript version of the module (MikroORM metadata, joiner configs,
// linkable, relation map of the loaded application).
type upstreamDump struct {
	Metas         []json.RawMessage            `json:"metas"`
	JoinerConfig  json.RawMessage              `json:"joinerConfig"`
	Linkable      map[string]map[string]any    `json:"linkable"`
	JoinerConfigs []json.RawMessage            `json:"joinerConfigs"`
	RelationMap   map[string]map[string]string `json:"relationMap"`
}

func loadDump(t *testing.T) upstreamDump {
	t.Helper()
	raw, err := os.ReadFile("testdata/upstream.json")
	if err != nil {
		t.Fatal(err)
	}
	var d upstreamDump
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func canonical(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var x any
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := dec.Decode(&x); err != nil {
		t.Fatal(err)
	}
	out, _ := json.Marshal(x)
	return string(out)
}

func configJSON(t *testing.T, c *joiner.ModuleJoinerConfig, withSchema bool) string {
	t.Helper()
	c = c.Clone()
	if !withSchema {
		c.Schema = ""
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestModelsMatchUpstream(t *testing.T) {
	d := loadDump(t)
	for i, m := range Models {
		var want any
		if err := json.Unmarshal(d.Metas[i], &want); err != nil {
			t.Fatal(err)
		}
		// decode through the Go type (drops what the Go metadata omits)
		var wantMeta dml.EntityMeta
		if err := json.Unmarshal(d.Metas[i], &wantMeta); err != nil {
			t.Fatal(err)
		}
		if a, b := canonical(t, wantMeta), canonical(t, m.Meta); a != b {
			t.Errorf("%s meta:\nwant %s\ngot  %s", m.Name, a, b)
		}
	}
}

func TestJoinerConfigAndLinkableMatchUpstream(t *testing.T) {
	d := loadDump(t)
	want := joiner.MustParseModuleJoinerConfig(string(d.JoinerConfig))
	if a, b := configJSON(t, want, true), configJSON(t, Module.JoinerConfig, true); a != b {
		t.Errorf("joiner config:\nwant %s\ngot  %s", a, b)
	}
	got := map[string]map[string]any{}
	for field, cfg := range Module.Linkable {
		got[field] = map[string]any{}
		for _, l := range cfg.Linkables {
			got[field][l.PrimaryKey] = l
		}
	}
	if a, b := canonical(t, d.Linkable), canonical(t, got); a != b {
		t.Errorf("linkable:\nwant %s\ngot  %s", a, b)
	}
}

func TestQueryConfigMatchesUpstream(t *testing.T) {
	d := loadDump(t)
	utilsmodulessdk.ClearCustomLinks()
	t.Cleanup(utilsmodulessdk.ClearCustomLinks)
	if err := DefineLinks(); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefineConfig(config.InputConfig{Modules: []config.ModuleConfig{{Resolve: "./src/modules/brand", Module: &Module.ModuleExports}}})
	qc, err := loaders.BuildQueryConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var want, got []string
	for _, raw := range d.JoinerConfigs {
		want = append(want, configJSON(t, joiner.MustParseModuleJoinerConfig(string(raw)), false))
	}
	for _, c := range qc.JoinerConfigs {
		got = append(got, configJSON(t, c, false))
	}
	sort.Strings(want)
	sort.Strings(got)
	if !reflect.DeepEqual(want, got) {
		for _, w := range want {
			found := false
			for _, g := range got {
				found = found || g == w
			}
			if !found {
				t.Errorf("missing joiner config %s", w)
			}
		}
		for _, g := range got {
			found := false
			for _, w := range want {
				found = found || g == w
			}
			if !found {
				t.Errorf("unexpected joiner config %s", g)
			}
		}
	}
	if a, b := canonical(t, d.RelationMap), canonical(t, qc.RelationMap); a != b {
		for e, props := range d.RelationMap {
			if !reflect.DeepEqual(props, qc.RelationMap[e]) {
				t.Errorf("relation map %s:\nwant %v\ngot  %v", e, props, qc.RelationMap[e])
			}
		}
		for e := range qc.RelationMap {
			if _, ok := d.RelationMap[e]; !ok {
				t.Errorf("extra relation map entity %s", e)
			}
		}
	}
}
