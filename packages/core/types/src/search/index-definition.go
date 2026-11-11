// Package search mirrors @medusajs/types/src/search: search index
// definitions, queries, results and the provider contract.
package search

import (
	"context"
	"log/slog"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Field kinds (SearchFieldKind).
const (
	KindText    = "text"
	KindKeyword = "keyword"
	KindInteger = "integer"
	KindFloat   = "float"
	KindBoolean = "boolean"
	KindDate    = "date"
	KindGeo     = "geo"
	KindObject  = "object"
	KindVector  = "vector"
)

// SearchableOptions mirrors the object form of `searchable`.
type SearchableOptions struct {
	// Weight is nil when undefined.
	Weight *float64
}

// FacetableOptions mirrors the object form of `facetable`.
type FacetableOptions struct {
	// Types is nil when undefined.
	Types []string
}

// FieldDefinition mirrors SearchFieldDefinition. Optional members are nil
// when undefined.
type FieldDefinition struct {
	Type  string
	Array *bool
	// Searchable is nil, a bool or *SearchableOptions.
	Searchable any
	Filterable *bool
	Sortable   *bool
	// Facetable is nil, a bool or *FacetableOptions.
	Facetable       any
	Retrievable     *bool
	Dimensions      *int
	Embed           *bool
	ProviderOptions *dal.Record
	// Fields are the nested fields of an object field (nil otherwise).
	Fields Fields
}

// Field is one named field of a definition (definitions keep their
// declaration order, like the keys of a JavaScript object).
type Field struct {
	Name string
	Def  *FieldDefinition
}

// Fields is an ordered field map.
type Fields []Field

// Get returns the definition of a field (nil when absent).
func (f Fields) Get(name string) *FieldDefinition {
	for _, e := range f {
		if e.Name == name {
			return e.Def
		}
	}
	return nil
}

// ToRecord writes the field like the plain definition `toFields()`
// compiles (metadataToFieldDefinition key order).
func (d *FieldDefinition) ToRecord() *dal.Record {
	r := dal.NewRecord()
	r.Set("type", d.Type)
	if d.Array != nil {
		r.Set("array", *d.Array)
	}
	if d.Searchable != nil {
		r.Set("searchable", optionValue(d.Searchable))
	}
	if d.Filterable != nil {
		r.Set("filterable", *d.Filterable)
	}
	if d.Sortable != nil {
		r.Set("sortable", *d.Sortable)
	}
	if d.Facetable != nil {
		r.Set("facetable", optionValue(d.Facetable))
	}
	if d.Retrievable != nil {
		r.Set("retrievable", *d.Retrievable)
	}
	if d.Dimensions != nil {
		r.Set("dimensions", float64(*d.Dimensions))
	}
	if d.Embed != nil {
		r.Set("embed", *d.Embed)
	}
	if d.ProviderOptions != nil {
		r.Set("provider_options", d.ProviderOptions)
	}
	if d.Fields != nil {
		r.Set("fields", d.Fields.ToRecord())
	}
	return r
}

// ToRecord writes the fields as a plain object.
func (f Fields) ToRecord() *dal.Record {
	r := dal.NewRecord()
	for _, e := range f {
		r.Set(e.Name, e.Def.ToRecord())
	}
	return r
}

func optionValue(v any) any {
	switch t := v.(type) {
	case *SearchableOptions:
		r := dal.NewRecord()
		if t.Weight != nil {
			r.Set("weight", *t.Weight)
		}
		return r
	case *FacetableOptions:
		r := dal.NewRecord()
		if t.Types != nil {
			types := make([]any, len(t.Types))
			for i, s := range t.Types {
				types[i] = s
			}
			r.Set("types", types)
		}
		return r
	}
	return v
}

// TypoTolerance mirrors SearchIndexSettings.typo_tolerance.
type TypoTolerance struct {
	Enabled                *bool
	MinWordSizeForOneTypo  *float64
	MinWordSizeForTwoTypos *float64
	DisabledOnAttributes   []string
}

// Settings mirrors SearchIndexSettings.
type Settings struct {
	TypoTolerance     *TypoTolerance
	DistinctAttribute string
	ProviderOptions   *dal.Record
}

// ToRecord writes the settings as a plain object (undefined members left
// out).
func (s *Settings) ToRecord() *dal.Record {
	r := dal.NewRecord()
	if s == nil {
		return r
	}
	if t := s.TypoTolerance; t != nil {
		tr := dal.NewRecord()
		if t.Enabled != nil {
			tr.Set("enabled", *t.Enabled)
		}
		if t.MinWordSizeForOneTypo != nil {
			tr.Set("min_word_size_for_one_typo", *t.MinWordSizeForOneTypo)
		}
		if t.MinWordSizeForTwoTypos != nil {
			tr.Set("min_word_size_for_two_typos", *t.MinWordSizeForTwoTypos)
		}
		if t.DisabledOnAttributes != nil {
			attrs := make([]any, len(t.DisabledOnAttributes))
			for i, a := range t.DisabledOnAttributes {
				attrs[i] = a
			}
			tr.Set("disabled_on_attributes", attrs)
		}
		r.Set("typo_tolerance", tr)
	}
	if s.DistinctAttribute != "" {
		r.Set("distinct_attribute", s.DistinctAttribute)
	}
	if s.ProviderOptions != nil {
		r.Set("provider_options", s.ProviderOptions)
	}
	return r
}

// Mutation mirrors SearchMutation: an upsert of documents or a delete by
// filters.
type Mutation struct {
	Action    string // "upsert" | "delete"
	Documents []*dal.Record
	Filters   *dal.Record
}

// Event is the event a definition consumes (Event: { name, data, metadata }).
type Event struct {
	Name     string
	Data     any
	Metadata *dal.Record
}

// Container mirrors SearchContainer: what seed and consume read through.
// Query is the application's Query (*query.Query).
type Container struct {
	Query  any
	Logger *slog.Logger
}

// IngestionContext mirrors SearchIngestionContext.
type IngestionContext struct {
	Container Container
	Index     *ResolvedDefinition
}

// Catchup mirrors SearchSeedContext.catchup.
type Catchup struct {
	Since time.Time
}

// SeedContext mirrors SearchSeedContext.
type SeedContext struct {
	IngestionContext
	Filters *dal.Record
	LastKey *string
	Catchup *Catchup
}

// SeedFunc mirrors `seed`: it yields batches of mutations (yield returns
// the error that stops the iteration).
type SeedFunc func(ctx context.Context, sc SeedContext, yield func([]Mutation) error) error

// ConsumeFunc mirrors `consume`.
type ConsumeFunc func(ctx context.Context, event Event, ic IngestionContext) ([]Mutation, error)

// Definition mirrors SearchIndexDefinition (fields already compiled to
// plain definitions).
type Definition struct {
	Name   string
	Entity string
	// PrimaryKey is "" when undefined (defaults to "id").
	PrimaryKey string
	// Provider is "" when undefined (the default provider).
	Provider string
	Fields   Fields
	Settings *Settings
	Events   []string
	Consume  ConsumeFunc
	Seed     SeedFunc
}

// ResolvedDefinition mirrors ResolvedSearchIndexDefinition.
type ResolvedDefinition struct {
	Definition
	DefinitionHash string
	PhysicalName   string
}

// WithPhysicalName returns a copy pointing at another physical index.
func (d *ResolvedDefinition) WithPhysicalName(name string) *ResolvedDefinition {
	c := *d
	c.PhysicalName = name
	return &c
}
