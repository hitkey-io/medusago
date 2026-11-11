// Package utils mirrors @medusajs/search-postgres/src/utils.
package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"

	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Engines (PostgresSearchEngine).
const (
	EngineNative   = "native"
	EngineLakebase = "lakebase"
)

// CatalogTable mirrors CATALOG_TABLE.
const CatalogTable = "search_postgres_index"

// PlannedField mirrors PlannedField.
type PlannedField struct {
	Path       string
	Kind       string // text | keyword | number | boolean | geo | vector
	IsArray    bool
	IsDate     bool
	Dimensions *int
	Field      *searchtypes.FieldDefinition
}

// IndexPlan mirrors IndexPlan (Fields keeps the insertion order of the Map).
type IndexPlan struct {
	Order      []string
	Fields     map[string]*PlannedField
	Searchable []string
	Vectors    []string
	PrimaryKey string
	SchemaHash string
}

// Field returns a planned field (nil when absent).
func (p *IndexPlan) Field(path string) *PlannedField {
	return p.Fields[path]
}

// List returns the planned fields in order.
func (p *IndexPlan) List() []*PlannedField {
	out := make([]*PlannedField, len(p.Order))
	for i, path := range p.Order {
		out[i] = p.Fields[path]
	}
	return out
}

func (p *IndexPlan) set(f *PlannedField) {
	if _, ok := p.Fields[f.Path]; !ok {
		p.Order = append(p.Order, f.Path)
	}
	p.Fields[f.Path] = f
}

func fail(message string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, message)
}

// IsSearchable mirrors isSearchable.
func IsSearchable(field *searchtypes.FieldDefinition) bool {
	if b, ok := field.Searchable.(bool); ok {
		return b
	}
	_, ok := field.Searchable.(*searchtypes.SearchableOptions)
	return ok
}

// IsFacetable mirrors isFacetable.
func IsFacetable(field *searchtypes.FieldDefinition) bool {
	if b, ok := field.Facetable.(bool); ok {
		return b
	}
	_, ok := field.Facetable.(*searchtypes.FacetableOptions)
	return ok
}

func fieldKind(field *searchtypes.FieldDefinition) (string, error) {
	switch field.Type {
	case "text":
		return "text", nil
	case "keyword":
		return "keyword", nil
	case "integer", "float", "date":
		return "number", nil
	case "boolean":
		return "boolean", nil
	case "geo":
		return "geo", nil
	case "vector":
		return "vector", nil
	}
	return "", fail(fmt.Sprintf(`The postgres search provider does not support fields of type "%s"`, field.Type))
}

func isTrue(b *bool) bool { return b != nil && *b }

// AssertIndexSupported mirrors assertIndexSupported.
func AssertIndexSupported(definition *searchtypes.ResolvedDefinition, engine string, logger *slog.Logger) error {
	supportsVectors := engine == EngineLakebase
	var ignoredVectors []string
	var walk func(group searchtypes.Fields, prefix string) error
	walk = func(group searchtypes.Fields, prefix string) error {
		for _, e := range group {
			field := e.Def
			path := e.Name
			if prefix != "" {
				path = prefix + "." + e.Name
			}
			if field.Type == "vector" {
				if !supportsVectors {
					ignoredVectors = append(ignoredVectors, path)
					continue
				}
				if field.Dimensions == nil || *field.Dimensions < 1 {
					return fail(fmt.Sprintf(`Vector field "%s" must declare a positive "dimensions" value`, path))
				}
				if isTrue(field.Array) {
					return fail(fmt.Sprintf(`Vector field "%s" cannot be an array`, path))
				}
			}
			if field.Type == "geo" {
				return fail(fmt.Sprintf(`The postgres search provider does not support geo fields ("%s")`, path))
			}
			if field.Type == "object" && field.Fields != nil {
				if err := walk(field.Fields, path); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(definition.Fields, ""); err != nil {
		return err
	}
	if s := definition.Settings; s != nil && s.TypoTolerance != nil && len(s.TypoTolerance.DisabledOnAttributes) > 0 {
		return fail("The postgres search provider matches typos across all searchable fields at once, so it does not support settings.typo_tolerance.disabled_on_attributes")
	}
	if len(ignoredVectors) > 0 && logger != nil {
		quoted := make([]string, len(ignoredVectors))
		for i, p := range ignoredVectors {
			quoted[i] = `"` + p + `"`
		}
		logger.Warn(fmt.Sprintf(`The postgres search provider does not support vector search on the "%s" engine. Ignoring vector field(s) %s on search index "%s". Use Medusa Search to enable vector search.`, engine, strings.Join(quoted, ", "), definition.Name))
	}
	return nil
}

func recordGet(r *dal.Record, key string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(key)
	return v
}

// AssertQuerySupported mirrors assertQuerySupported.
func AssertQuerySupported(query *searchtypes.ProviderQuery, engine string) error {
	options := query.SearchOptions
	q := ""
	if query.Q != nil {
		q = strings.TrimSpace(*query.Q)
	}
	if truthy(recordGet(options, "highlight")) && q != "" {
		return fail("The postgres search provider does not support highlighting")
	}
	if truthy(recordGet(options, "vector")) && engine != EngineLakebase {
		return fail(`Vector search requires engine: "lakebase". The native engine does not support search_options.vector.`)
	}
	if locales, ok := recordGet(options, "locales").([]any); ok && len(locales) > 0 {
		return fail("The postgres search provider does not support query-time locales; configure `language` on the provider options instead")
	}
	if query.Pagination != nil && query.Pagination.Has("cursor") {
		if v, _ := query.Pagination.Get("cursor"); v != nil {
			return fail("The postgres search provider does not support cursor pagination")
		}
	}
	return nil
}

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	}
	return true
}

// KeywordTsQuerySql mirrors keywordTsQuerySql.
func KeywordTsQuerySql(tsConfig string, matchStrategy string) string {
	parsed := fmt.Sprintf("plainto_tsquery('%s', ?)", tsConfig)
	switch matchStrategy {
	case "any":
		return fmt.Sprintf("replace(%s::text, ' & ', ' | ')::tsquery", parsed)
	case "last":
		return fmt.Sprintf(`regexp_replace(%s::text, '''([^'']+)''$', '''\1'':*')::tsquery`, parsed)
	}
	return parsed
}

// BuildIndexPlan mirrors buildIndexPlan.
func BuildIndexPlan(definition *searchtypes.ResolvedDefinition, engine string) (*IndexPlan, error) {
	plan := &IndexPlan{Fields: map[string]*PlannedField{}, PrimaryKey: definition.PrimaryKey, Searchable: []string{}, Vectors: []string{}}
	supportsVectors := engine == EngineLakebase
	var walk func(group searchtypes.Fields, prefix string, inArray bool) error
	walk = func(group searchtypes.Fields, prefix string, inArray bool) error {
		for _, e := range group {
			field := e.Def
			path := e.Name
			if prefix != "" {
				path = prefix + "." + e.Name
			}
			isArray := inArray || isTrue(field.Array)
			if field.Type == "object" {
				if field.Fields != nil {
					if err := walk(field.Fields, path, isArray); err != nil {
						return err
					}
				}
				continue
			}
			if field.Type == "vector" && !supportsVectors {
				continue
			}
			kind, err := fieldKind(field)
			if err != nil {
				return err
			}
			plan.set(&PlannedField{Path: path, Kind: kind, IsArray: isArray, IsDate: field.Type == "date", Dimensions: field.Dimensions, Field: field})
			if kind == "vector" {
				plan.Vectors = append(plan.Vectors, path)
				continue
			}
			if IsSearchable(field) {
				plan.Searchable = append(plan.Searchable, path)
			}
		}
		return nil
	}
	if err := walk(definition.Fields, "", false); err != nil {
		return nil, err
	}

	fields := make([]any, 0, len(plan.Order))
	for _, planned := range plan.List() {
		f := dal.NewRecord()
		f.Set("path", planned.Path)
		f.Set("kind", planned.Kind)
		f.Set("is_array", planned.IsArray)
		f.Set("is_date", planned.IsDate)
		if planned.Dimensions != nil {
			f.Set("dimensions", float64(*planned.Dimensions))
		}
		if planned.Field.Embed != nil {
			f.Set("embed", *planned.Field.Embed)
		}
		f.Set("searchable", IsSearchable(planned.Field))
		f.Set("filterable", isTrue(planned.Field.Filterable))
		f.Set("sortable", isTrue(planned.Field.Sortable))
		f.Set("facetable", IsFacetable(planned.Field))
		fields = append(fields, f)
	}
	payload := dal.NewRecord()
	payload.Set("primary_key", definition.PrimaryKey)
	payload.Set("fields", fields)
	payload.Set("searchable", sortedAny(plan.Searchable))
	payload.Set("vectors", sortedAny(plan.Vectors))
	raw, err := dal.MarshalValue(payload)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(raw)
	plan.SchemaHash = hex.EncodeToString(sum[:])
	return plan, nil
}

func sortedAny(values []string) []any {
	s := append([]string{}, values...)
	sort.Strings(s)
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}

// SameSchema mirrors sameSchema.
func SameSchema(a, b *IndexPlan) bool { return a.SchemaHash == b.SchemaHash }

// WeightLabel mirrors weightLabel.
func WeightLabel(searchable any) string {
	weight := 1.0
	if o, ok := searchable.(*searchtypes.SearchableOptions); ok && o.Weight != nil {
		weight = *o.Weight
	}
	switch {
	case weight >= 3:
		return "A"
	case weight >= 2:
		return "B"
	case weight > 1:
		return "C"
	}
	return "D"
}

var (
	unsafeChars = regexp.MustCompile(`[^a-z0-9_]`)
	underscores = regexp.MustCompile(`_+`)
	edgeUnders  = regexp.MustCompile(`^_|_$`)
)

// TableNameForIndex mirrors tableNameForIndex.
func TableNameForIndex(physicalName string) (string, error) {
	safe := strings.ToLower(physicalName)
	safe = unsafeChars.ReplaceAllString(safe, "_")
	safe = underscores.ReplaceAllString(safe, "_")
	safe = edgeUnders.ReplaceAllString(safe, "")
	if safe == "" {
		return "", fail(fmt.Sprintf(`Cannot derive a table name from index "%s"`, physicalName))
	}
	return "search_pg_" + safe, nil
}

// Bm25IndexName mirrors bm25IndexName.
func Bm25IndexName(table string) string { return table + "_bm25" }

// VectorColumnName mirrors vectorColumnName.
func VectorColumnName(path string) string {
	safe := strings.ToLower(path)
	safe = unsafeChars.ReplaceAllString(safe, "_")
	safe = underscores.ReplaceAllString(safe, "_")
	return "v_" + safe
}

// SerializePlan mirrors serializePlan.
func SerializePlan(plan *IndexPlan) (string, error) {
	out := dal.NewRecord()
	out.Set("primary_key", plan.PrimaryKey)
	out.Set("searchable", toAny(plan.Searchable))
	out.Set("vectors", toAny(plan.Vectors))
	out.Set("schema_hash", plan.SchemaHash)
	fields := make([]any, 0, len(plan.Order))
	for _, planned := range plan.List() {
		f := dal.NewRecord()
		f.Set("path", planned.Path)
		f.Set("kind", planned.Kind)
		f.Set("is_array", planned.IsArray)
		f.Set("is_date", planned.IsDate)
		if planned.Dimensions != nil {
			f.Set("dimensions", float64(*planned.Dimensions))
		}
		f.Set("field", planned.Field.ToRecord())
		fields = append(fields, f)
	}
	out.Set("fields", fields)
	raw, err := dal.MarshalValue(out)
	return string(raw), err
}

func toAny(values []string) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

func toStrings(v any) []string {
	arr, _ := v.([]any)
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		s, _ := e.(string)
		out = append(out, s)
	}
	return out
}

// DeserializePlan mirrors deserializePlan (the catalog's jsonb plan).
func DeserializePlan(raw any) *IndexPlan {
	data, _ := raw.(*dal.Record)
	if s, ok := raw.(string); ok {
		v, _ := dal.DecodeOrderedJSON([]byte(s))
		data, _ = v.(*dal.Record)
	}
	plan := &IndexPlan{Fields: map[string]*PlannedField{}}
	if data == nil {
		return plan
	}
	entries, _ := recordGet(data, "fields").([]any)
	for _, e := range entries {
		entry, _ := e.(*dal.Record)
		if entry == nil {
			continue
		}
		path, _ := recordGet(entry, "path").(string)
		kind, _ := recordGet(entry, "kind").(string)
		isArray, _ := recordGet(entry, "is_array").(bool)
		isDate, _ := recordGet(entry, "is_date").(bool)
		var dims *int
		if d, ok := recordGet(entry, "dimensions").(float64); ok {
			n := int(d)
			dims = &n
		}
		field, _ := recordGet(entry, "field").(*dal.Record)
		plan.set(&PlannedField{Path: path, Kind: kind, IsArray: isArray, IsDate: isDate, Dimensions: dims, Field: ParseFieldDefinition(field)})
	}
	plan.PrimaryKey, _ = recordGet(data, "primary_key").(string)
	plan.Searchable = toStrings(recordGet(data, "searchable"))
	if v, ok := data.Get("vectors"); ok && v != nil {
		plan.Vectors = toStrings(v)
	} else {
		plan.Vectors = []string{}
		for _, f := range plan.List() {
			if f.Kind == "vector" {
				plan.Vectors = append(plan.Vectors, f.Path)
			}
		}
	}
	plan.SchemaHash, _ = recordGet(data, "schema_hash").(string)
	return plan
}

func boolPtr(v any) *bool {
	b, ok := v.(bool)
	if !ok {
		return nil
	}
	return &b
}

// ParseFieldDefinition reads a plain field definition (stored plans).
func ParseFieldDefinition(r *dal.Record) *searchtypes.FieldDefinition {
	def := &searchtypes.FieldDefinition{}
	if r == nil {
		return def
	}
	def.Type, _ = recordGet(r, "type").(string)
	def.Array = boolPtr(recordGet(r, "array"))
	switch v := recordGet(r, "searchable").(type) {
	case bool:
		def.Searchable = v
	case *dal.Record:
		o := &searchtypes.SearchableOptions{}
		if w, ok := recordGet(v, "weight").(float64); ok {
			o.Weight = &w
		}
		def.Searchable = o
	}
	def.Filterable = boolPtr(recordGet(r, "filterable"))
	def.Sortable = boolPtr(recordGet(r, "sortable"))
	switch v := recordGet(r, "facetable").(type) {
	case bool:
		def.Facetable = v
	case *dal.Record:
		o := &searchtypes.FacetableOptions{}
		if t, ok := v.Get("types"); ok && t != nil {
			o.Types = toStrings(t)
		}
		def.Facetable = o
	}
	def.Retrievable = boolPtr(recordGet(r, "retrievable"))
	if d, ok := recordGet(r, "dimensions").(float64); ok {
		n := int(d)
		def.Dimensions = &n
	}
	def.Embed = boolPtr(recordGet(r, "embed"))
	def.ProviderOptions, _ = recordGet(r, "provider_options").(*dal.Record)
	if fields, ok := recordGet(r, "fields").(*dal.Record); ok {
		def.Fields = searchtypes.Fields{}
		for _, k := range fields.Keys() {
			v, _ := fields.Get(k)
			nested, _ := v.(*dal.Record)
			def.Fields = append(def.Fields, searchtypes.Field{Name: k, Def: ParseFieldDefinition(nested)})
		}
	}
	return def
}

// TextSearchConfigName mirrors textSearchConfigName.
func TextSearchConfigName(language string) string {
	return "medusa_search_" + language
}

// WordSimilarityCall mirrors wordSimilarityCall.
func WordSimilarityCall(querySql, textSql string) string {
	return fmt.Sprintf("word_similarity(%s, %s)", querySql, textSql)
}
