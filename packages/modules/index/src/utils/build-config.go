package utils

import (
	"encoding/json"
	"fmt"
)

// SchemaObjectRepresentationPropertiesToOmit mirrors
// schemaObjectRepresentationPropertiesToOmit (types/index.ts).
var SchemaObjectRepresentationPropertiesToOmit = []string{
	"_schemaPropertiesMap",
	"_serviceNameModuleConfigMap",
	"JSON",
	"DateTime",
	"Date",
	"Time",
}

// Relationship is the part of a link joiner relationship the index uses.
type Relationship struct {
	ServiceName string `json:"serviceName"`
	ForeignKey  string `json:"foreignKey"`
}

// ModuleConfig is the part of the joiner config of an indexed entity's
// module the index uses.
type ModuleConfig struct {
	ServiceName   string         `json:"serviceName"`
	IsLink        bool           `json:"isLink"`
	PrimaryKeys   []string       `json:"primaryKeys"`
	Relationships []Relationship `json:"relationships"`
}

// Parent mirrors SchemaObjectEntityRepresentation["parents"][0].
type Parent struct {
	Ref             *SchemaObjectEntityRepresentation
	InSchemaRef     *SchemaObjectEntityRepresentation
	TargetProp      string
	InverseSideProp string
	IsList          bool
	IsInverse       bool
}

// SchemaObjectEntityRepresentation mirrors SchemaObjectEntityRepresentation.
type SchemaObjectEntityRepresentation struct {
	Entity       string
	Alias        string
	Fields       []string
	Listeners    []string
	ModuleConfig ModuleConfig
	Parents      []Parent
}

// SchemaPropertiesMapEntry mirrors SchemaPropertiesMap[0].
type SchemaPropertiesMapEntry struct {
	ShortCutOf string
	Ref        *SchemaObjectEntityRepresentation
	IsInverse  bool
	IsList     bool
}

// SchemaObjectRepresentation mirrors SchemaObjectRepresentation: the
// entities keyed by type name (in schema order), _schemaPropertiesMap and
// the linkable keys of _serviceNameModuleConfigMap, plus the GraphQL field
// types of the executable schema (entitiesMap) the query builder casts with.
type SchemaObjectRepresentation struct {
	// Keys lists the entity type names in insertion order.
	Keys     []string
	Entities map[string]*SchemaObjectEntityRepresentation
	// PropertiesMap is _schemaPropertiesMap.
	PropertiesMap map[string]*SchemaPropertiesMapEntry
	// LinkableKeys is _serviceNameModuleConfigMap[service].linkableKeys.
	LinkableKeys map[string]map[string]string
	// Types maps a GraphQL type to its fields and their type (toString()).
	Types map[string]map[string]string
}

type rawRepresentation struct {
	Entities []struct {
		Key          string       `json:"key"`
		Entity       string       `json:"entity"`
		Alias        *string      `json:"alias"`
		Fields       []string     `json:"fields"`
		Listeners    []string     `json:"listeners"`
		ModuleConfig ModuleConfig `json:"moduleConfig"`
		Parents      []struct {
			Ref             string  `json:"ref"`
			InSchemaRef     *string `json:"inSchemaRef"`
			TargetProp      string  `json:"targetProp"`
			InverseSideProp string  `json:"inverseSideProp"`
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

// ParseSchemaObjectRepresentation decodes an extracted representation.
func ParseSchemaObjectRepresentation(data string) (*SchemaObjectRepresentation, error) {
	var raw rawRepresentation
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
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
		out.Entities[e.Key] = &SchemaObjectEntityRepresentation{
			Entity:       e.Entity,
			Alias:        strOrEmpty(e.Alias),
			Fields:       append([]string(nil), e.Fields...),
			Listeners:    append([]string(nil), e.Listeners...),
			ModuleConfig: e.ModuleConfig,
		}
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
			parent := Parent{Ref: r, TargetProp: p.TargetProp, InverseSideProp: p.InverseSideProp, IsList: truthy(p.IsList), IsInverse: truthy(p.IsInverse)}
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
		out.PropertiesMap[p.Path] = &SchemaPropertiesMapEntry{ShortCutOf: strOrEmpty(p.ShortCutOf), Ref: r, IsInverse: truthy(p.IsInverse), IsList: truthy(p.IsList)}
	}
	return out, nil
}

// BuildSchemaObjectRepresentation mirrors buildSchemaObjectRepresentation
// for defaultSchema. The representation is extracted from upstream (the
// GraphQL schema merge over the module joiner configs is not recomputed, so
// a custom `schema` module option is not supported). Each call returns a
// fresh copy: the configuration check sorts the entity fields in place,
// like upstream.
func BuildSchemaObjectRepresentation() (*SchemaObjectRepresentation, error) {
	return ParseSchemaObjectRepresentation(defaultSchemaObjectRepresentationJSON)
}

// AllFields returns the fields of every entity (allSchemaFields).
func (s *SchemaObjectRepresentation) AllFields() map[string]bool {
	out := map[string]bool{}
	for _, k := range s.Keys {
		for _, f := range s.Entities[k].Fields {
			out[f] = true
		}
	}
	return out
}
