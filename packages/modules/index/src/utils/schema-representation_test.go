package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

// rawRepresentation is the shape testdata/index-schemas/*.json are dumped
// in from upstream (integration-tests/http-diff/testapp/dump-index-schema.js).
type rawRepresentation struct {
	Entities []struct {
		Key          string        `json:"key"`
		Entity       string        `json:"entity"`
		Alias        *string       `json:"alias"`
		Fields       []string      `json:"fields"`
		Listeners    []string      `json:"listeners"`
		ModuleConfig *ModuleConfig `json:"moduleConfig"`
		Parents      []struct {
			Ref             string  `json:"ref"`
			InSchemaRef     *string `json:"inSchemaRef"`
			TargetProp      string  `json:"targetProp"`
			InverseSideProp *string `json:"inverseSideProp"`
			IsList          *bool   `json:"isList"`
			IsInverse       *bool   `json:"isInverse"`
		} `json:"parents"`
	} `json:"entities"`
	PropertiesMap []struct {
		Path       string  `json:"path"`
		Ref        string  `json:"ref"`
		ShortCutOf *string `json:"shortCutOf"`
		IsInverse  *bool   `json:"isInverse"`
		IsList     *bool   `json:"isList"`
	} `json:"propertiesMap"`
	LinkableKeys map[string]map[string]string `json:"linkableKeys"`
	Types        map[string]map[string]string `json:"types"`
}

func truthy(b *bool) bool { return b != nil && *b }

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// loadSchemaObjectRepresentation decodes an upstream representation dump.
func loadSchemaObjectRepresentation(file string) (*SchemaObjectRepresentation, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var raw rawRepresentation
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := &SchemaObjectRepresentation{
		Entities:      map[string]*SchemaObjectEntityRepresentation{},
		PropertiesMap: map[string]*SchemaPropertiesMapEntry{},
		LinkableKeys:  raw.LinkableKeys,
		Types:         raw.Types,
	}
	for _, e := range raw.Entities {
		out.Keys = append(out.Keys, e.Key)
		rep := &SchemaObjectEntityRepresentation{
			Entity:    e.Entity,
			Alias:     strOrEmpty(e.Alias),
			Fields:    append([]string(nil), e.Fields...),
			Listeners: append([]string(nil), e.Listeners...),
		}
		if e.ModuleConfig != nil {
			rep.ModuleConfig = *e.ModuleConfig
		}
		out.Entities[e.Key] = rep
	}
	ref := func(name string) (*SchemaObjectEntityRepresentation, error) {
		r, ok := out.Entities[name]
		if !ok {
			return nil, fmt.Errorf("index schema: unknown entity %q", name)
		}
		return r, nil
	}
	for _, e := range raw.Entities {
		rep := out.Entities[e.Key]
		for _, p := range e.Parents {
			r, err := ref(p.Ref)
			if err != nil {
				return nil, err
			}
			parent := Parent{Ref: r, TargetProp: p.TargetProp, InverseSideProp: strOrEmpty(p.InverseSideProp), IsList: truthy(p.IsList), IsListUnset: p.IsList == nil, IsInverse: truthy(p.IsInverse)}
			if p.InSchemaRef != nil {
				if parent.InSchemaRef, err = ref(*p.InSchemaRef); err != nil {
					return nil, err
				}
			}
			rep.Parents = append(rep.Parents, parent)
		}
	}
	for _, p := range raw.PropertiesMap {
		r, err := ref(p.Ref)
		if err != nil {
			return nil, err
		}
		out.PropertiesKeys = append(out.PropertiesKeys, p.Path)
		out.PropertiesMap[p.Path] = &SchemaPropertiesMapEntry{ShortCutOf: strOrEmpty(p.ShortCutOf), Ref: r, IsInverse: truthy(p.IsInverse), IsList: truthy(p.IsList)}
	}
	return out, nil
}
