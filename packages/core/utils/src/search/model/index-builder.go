// Package model mirrors @medusajs/utils/src/search/model: the search fields
// DSL (`search.define({ ... })`).
package model

import (
	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Property mirrors a search field property (BaseSearchProperty and its
// subclasses). Every modifier returns the property, like the TypeScript
// builder; the ones a kind does not offer upstream are a compile error
// there and are simply not used here.
type Property struct {
	def searchtypes.FieldDefinition
}

func newProperty(kind string) *Property {
	return &Property{def: searchtypes.FieldDefinition{Type: kind}}
}

func boolArg(v []bool) *bool {
	b := true
	if len(v) > 0 {
		b = v[0]
	}
	return &b
}

// Retrievable mirrors retrievable(value = true).
func (p *Property) Retrievable(v ...bool) *Property {
	p.def.Retrievable = boolArg(v)
	return p
}

// ProviderOptions mirrors providerOptions(options).
func (p *Property) ProviderOptions(options *dal.Record) *Property {
	p.def.ProviderOptions = options
	return p
}

// Filterable mirrors filterable(value = true).
func (p *Property) Filterable(v ...bool) *Property {
	p.def.Filterable = boolArg(v)
	return p
}

// Sortable mirrors sortable(value = true).
func (p *Property) Sortable(v ...bool) *Property {
	p.def.Sortable = boolArg(v)
	return p
}

// Facetable mirrors facetable(value = true).
func (p *Property) Facetable(v ...bool) *Property {
	p.def.Facetable = *boolArg(v)
	return p
}

// FacetableTypes mirrors facetable({ types }) (range/stats facet kinds).
func (p *Property) FacetableTypes(types ...string) *Property {
	p.def.Facetable = &searchtypes.FacetableOptions{Types: types}
	return p
}

// Array mirrors array().
func (p *Property) Array() *Property {
	b := true
	p.def.Array = &b
	return p
}

// Searchable mirrors searchable(value = true).
func (p *Property) Searchable(v ...bool) *Property {
	p.def.Searchable = *boolArg(v)
	return p
}

// SearchableWeight mirrors searchable({ weight }).
func (p *Property) SearchableWeight(weight float64) *Property {
	p.def.Searchable = &searchtypes.SearchableOptions{Weight: &weight}
	return p
}

// Embed mirrors embed() of a vector property.
func (p *Property) Embed() *Property {
	b := true
	p.def.Embed = &b
	return p
}

// F is one named property of a schema.
type F struct {
	Name     string
	Property *Property
}

// Field names a property (search.define key).
func Field(name string, property *Property) F { return F{Name: name, Property: property} }

// parse mirrors property.parse(name) + metadataToFieldDefinition.
func (p *Property) parse() *searchtypes.FieldDefinition {
	def := p.def
	return &def
}

// Define mirrors search.define({ ... }).toFields(): the plain field
// definitions of a schema, in declaration order.
func Define(fields ...F) searchtypes.Fields {
	out := make(searchtypes.Fields, 0, len(fields))
	for _, f := range fields {
		out = append(out, searchtypes.Field{Name: f.Name, Def: f.Property.parse()})
	}
	return out
}

// Keyword mirrors search.keyword().
func Keyword() *Property { return newProperty(searchtypes.KindKeyword) }

// Text mirrors search.text().
func Text() *Property { return newProperty(searchtypes.KindText) }

// Integer mirrors search.integer().
func Integer() *Property { return newProperty(searchtypes.KindInteger) }

// Float mirrors search.float().
func Float() *Property { return newProperty(searchtypes.KindFloat) }

// Boolean mirrors search.boolean().
func Boolean() *Property { return newProperty(searchtypes.KindBoolean) }

// Date mirrors search.date().
func Date() *Property { return newProperty(searchtypes.KindDate) }

// Geo mirrors search.geo().
func Geo() *Property { return newProperty(searchtypes.KindGeo) }

// Object mirrors search.object({ ... }).
func Object(fields ...F) *Property {
	p := newProperty(searchtypes.KindObject)
	p.def.Fields = Define(fields...)
	return p
}

// Vector mirrors search.vector(dimensions).
func Vector(dimensions int) *Property {
	p := newProperty(searchtypes.KindVector)
	p.def.Dimensions = &dimensions
	return p
}
