package graphql

import (
	"sort"
	"strings"
)

// Type is a type of a merged schema.
type Type struct {
	Kind   string
	Name   string
	Fields []*FieldDefinition
	Values []string
	Types  []string
	// Directives are the directives of the definition and its extensions
	// (same-name directives merged).
	Directives []*Directive
}

// Field returns the field with the given name.
func (t *Type) Field(name string) *FieldDefinition {
	for _, f := range t.Fields {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// Schema is a merged schema: its types in definition order.
type Schema struct {
	Types  []*Type
	byName map[string]*Type
}

// Type returns the type with the given name.
func (s *Schema) Type(name string) *Type {
	return s.byName[name]
}

// MergeTypeDefs mirrors mergeTypeDefs (@graphql-tools/merge) for the
// definitions Medusa schemas use: definitions and extensions of a type are
// merged into one type whose fields are the union of their fields, in order
// of appearance.
func MergeTypeDefs(docs ...*Document) *Schema {
	s := &Schema{byName: map[string]*Type{}}
	for _, doc := range docs {
		for _, d := range doc.Definitions {
			if d.Kind == KindDirective || d.Kind == KindSchema {
				continue
			}
			t, ok := s.byName[d.Name]
			if !ok {
				t = &Type{Kind: d.Kind, Name: d.Name}
				s.byName[d.Name] = t
				s.Types = append(s.Types, t)
			}
			if !d.Extension {
				t.Kind = d.Kind
			}
			for _, f := range d.Fields {
				if existing := t.Field(f.Name); existing != nil {
					// preventConflicts: a non-null redefinition makes the
					// field non-null
					if f.Type.NonNull && !existing.Type.NonNull {
						for i, x := range t.Fields {
							if x == existing {
								t.Fields[i] = &FieldDefinition{Name: existing.Name, Type: f.Type}
							}
						}
					}
					continue
				}
				t.Fields = append(t.Fields, f)
			}
			t.Directives = mergeDirectives(t.Directives, d.Directives)
			for _, v := range d.Values {
				if !containsString(t.Values, v) {
					t.Values = append(t.Values, v)
				}
			}
			for _, v := range d.Types {
				if !containsString(t.Types, v) {
					t.Types = append(t.Types, v)
				}
			}
		}
	}
	return s
}

// mergeDirectives mirrors mergeDirectives for non repeatable directives:
// the existing directives first, a directive already present gets the
// arguments it lacks and the list items its list arguments lack.
func mergeDirectives(existing, next []*Directive) []*Directive {
	out := existing
	for _, d := range next {
		var found *Directive
		for _, e := range out {
			if e.Name == d.Name {
				found = e
				break
			}
		}
		if found == nil {
			out = append(out, d)
			continue
		}
		merged := &Directive{Name: found.Name, Arguments: append([]*Argument(nil), found.Arguments...)}
		for _, a := range d.Arguments {
			idx := -1
			for i, b := range merged.Arguments {
				if b.Name == a.Name {
					idx = i
				}
			}
			if idx < 0 {
				merged.Arguments = append(merged.Arguments, a)
				continue
			}
			b := merged.Arguments[idx]
			if b.Value != nil && a.Value != nil && b.Value.Kind == ValueList && a.Value.Kind == ValueList {
				values := append([]*Value(nil), b.Value.Values...)
				for _, v := range a.Value.Values {
					dup := false
					for _, w := range values {
						if w.Kind == v.Kind && w.Value == v.Value {
							dup = true
						}
					}
					if !dup {
						values = append(values, v)
					}
				}
				merged.Arguments[idx] = &Argument{Name: b.Name, Value: &Value{Kind: ValueList, Values: values}}
			}
		}
		for i, e := range out {
			if e == found {
				out = append(append([]*Directive(nil), out[:i]...), append([]*Directive{merged}, out[i+1:]...)...)
				break
			}
		}
	}
	return out
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// IsObjectType reports whether the named type is an object type.
func (s *Schema) IsObjectType(name string) bool {
	t := s.byName[name]
	return t != nil && t.Kind == KindObject
}

// BuildSchema mirrors what the module runtime does with the joiner config
// schemas: the scalars Medusa defines, the schemas concatenated, cleaned
// (cleanGraphQLSchema) and merged (mergeTypeDefs, makeExecutableSchema).
func BuildSchema(schemas ...string) (*Schema, error) {
	doc, err := Parse(`
    scalar DateTime
    scalar JSON
    directive @enumValue(value: String) on ENUM_VALUE
  ` + strings.Join(schemas, "\n"))
	if err != nil {
		return nil, err
	}
	cleaned, _ := CleanGraphQLSchema(doc)
	return MergeTypeDefs(cleaned), nil
}

// introspectionRelations are the relations of the introspection object types
// every executable schema's type map contains.
var introspectionRelations = map[string]map[string]string{
	"__Directive":  {"args": "__InputValue"},
	"__EnumValue":  {},
	"__Field":      {"args": "__InputValue", "type": "__Type"},
	"__InputValue": {"type": "__Type"},
	"__Schema":     {"directives": "__Directive", "mutationType": "__Type", "queryType": "__Type", "subscriptionType": "__Type", "types": "__Type"},
	"__Type":       {"enumValues": "__EnumValue", "fields": "__Field", "inputFields": "__InputValue", "interfaces": "__Type", "ofType": "__Type", "possibleTypes": "__Type"},
}

// ExtractRelationsFromGQL mirrors extractRelationsFromGQL: for every object
// type of the schema type map (the introspection types included), its
// fields whose (unwrapped) type is an object type, mapped to that type.
func ExtractRelationsFromGQL(s *Schema) map[string]map[string]string {
	out := map[string]map[string]string{}
	for name, rel := range introspectionRelations {
		out[name] = map[string]string{}
		for k, v := range rel {
			out[name][k] = v
		}
	}
	for _, t := range s.Types {
		if t.Kind != KindObject || strings.HasPrefix(t.Name, "__") {
			continue
		}
		rel := map[string]string{}
		for _, f := range t.Fields {
			if name := f.Type.NamedType(); s.IsObjectType(name) {
				rel[f.Name] = name
			}
		}
		out[t.Name] = rel
	}
	return out
}

// TypeNames returns the names of the object types, sorted.
func (s *Schema) TypeNames() []string {
	var out []string
	for _, t := range s.Types {
		if t.Kind == KindObject {
			out = append(out, t.Name)
		}
	}
	sort.Strings(out)
	return out
}
