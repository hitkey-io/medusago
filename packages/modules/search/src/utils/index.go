// Package utils mirrors @medusajs/search/src/utils: definitions, query
// normalization, versions, migrations, seeding, ingestion and deletion.
package utils

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	searchutils "github.com/hitkey-io/medusago/packages/core/utils/src/search"
)

// SearchIndexState values.
const (
	StatePending  = "pending"
	StateBuilding = "building"
	StateReady    = "ready"
	StateError    = "error"
)

// SearchSyncStatus values.
const (
	SyncPending    = "pending"
	SyncProcessing = "processing"
	SyncDone       = "done"
	SyncFailed     = "failed"
	SyncCanceled   = "canceled"
)

const defaultTake = 20

// DefaultReindexBatchSize mirrors DEFAULT_REINDEX_BATCH_SIZE.
const DefaultReindexBatchSize = 100

// Indexes mirrors SearchIndexes: resolved definitions keyed by name, in
// registration order.
type Indexes struct {
	order  []string
	byName map[string]*searchtypes.ResolvedDefinition
}

// NewIndexes creates an empty registry.
func NewIndexes() *Indexes {
	return &Indexes{byName: map[string]*searchtypes.ResolvedDefinition{}}
}

// Get returns a definition.
func (i *Indexes) Get(name string) (*searchtypes.ResolvedDefinition, bool) {
	d, ok := i.byName[name]
	return d, ok
}

// Names returns the names in order.
func (i *Indexes) Names() []string { return append([]string{}, i.order...) }

// Values returns the definitions in order.
func (i *Indexes) Values() []*searchtypes.ResolvedDefinition {
	out := make([]*searchtypes.ResolvedDefinition, len(i.order))
	for k, name := range i.order {
		out[k] = i.byName[name]
	}
	return out
}

// Len returns the number of definitions.
func (i *Indexes) Len() int { return len(i.order) }

func (i *Indexes) set(d *searchtypes.ResolvedDefinition) {
	if _, ok := i.byName[d.Name]; !ok {
		i.order = append(i.order, d.Name)
	}
	i.byName[d.Name] = d
}

// ProviderRegistry is what the helpers need of SearchProviderService.
type ProviderRegistry interface {
	Retrieve(identifier string) (searchtypes.Provider, error)
}

// Locking mirrors the Locking Module's execute.
type Locking interface {
	Execute(ctx context.Context, key string, job func(ctx context.Context) error) error
}

// Context mirrors SearchIndexContext.
type Context struct {
	// Container returns the container seed and consume read through (the
	// application's Query is resolved when used).
	Container          func() searchtypes.Container
	Logger             Logger
	BatchSize          int
	Indexes            *Indexes
	Providers          ProviderRegistry
	IndexService       *utilsmodulessdk.InternalService
	VersionService     *utilsmodulessdk.InternalService
	SyncService        *utilsmodulessdk.InternalService
	Locking            Locking
	ActiveVersionCache *ActiveIndexVersionCache
}

// Logger is the logger the helpers write to.
type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

/* ----------------------------- field traversal ---------------------------- */

// FlatField mirrors FlatSearchField.
type FlatField struct {
	Path  string
	Field *searchtypes.FieldDefinition
}

// FlattenFields mirrors flattenFields.
func FlattenFields(fields searchtypes.Fields, prefix string) []FlatField {
	var flat []FlatField
	for _, e := range fields {
		path := e.Name
		if prefix != "" {
			path = prefix + "." + e.Name
		}
		flat = append(flat, FlatField{Path: path, Field: e.Def})
		if e.Def.Type == "object" && e.Def.Fields != nil {
			flat = append(flat, FlattenFields(e.Def.Fields, path)...)
		}
	}
	return flat
}

func buildFieldMap(fields searchtypes.Fields) map[string]FlatField {
	out := map[string]FlatField{}
	for _, f := range FlattenFields(fields, "") {
		out[f.Path] = f
	}
	return out
}

// IsSearchable mirrors isSearchable.
func IsSearchable(field *searchtypes.FieldDefinition) bool {
	if b, ok := field.Searchable.(bool); ok {
		return b
	}
	_, ok := field.Searchable.(*searchtypes.SearchableOptions)
	return ok
}

func defaultFacetKinds(field *searchtypes.FieldDefinition) []string {
	switch field.Type {
	case "integer", "float", "date":
		return []string{"range"}
	}
	return []string{"value"}
}

func containsString(list []string, v string) bool {
	for _, e := range list {
		if e == v {
			return true
		}
	}
	return false
}

// IsFacetable mirrors isFacetable.
func IsFacetable(field *searchtypes.FieldDefinition, kind string) bool {
	switch f := field.Facetable.(type) {
	case bool:
		return f && containsString(defaultFacetKinds(field), kind)
	case *searchtypes.FacetableOptions:
		types := f.Types
		if types == nil {
			types = defaultFacetKinds(field)
		}
		return containsString(types, kind)
	}
	return false
}

func isTrue(b *bool) bool { return b != nil && *b }

func isRetrievableField(field *searchtypes.FieldDefinition) bool {
	if field.Type == "object" {
		return false
	}
	if field.Type == "vector" {
		return isTrue(field.Retrievable)
	}
	return field.Retrievable == nil || *field.Retrievable
}

// ListRetrievablePaths mirrors listRetrievablePaths.
func ListRetrievablePaths(fields searchtypes.Fields) []string {
	out := []string{}
	for _, f := range FlattenFields(fields, "") {
		if isRetrievableField(f.Field) {
			out = append(out, f.Path)
		}
	}
	return out
}

// ListIndexedFields mirrors listIndexedFields.
func ListIndexedFields(fields searchtypes.Fields) []any {
	out := []any{}
	for _, f := range FlattenFields(fields, "") {
		if f.Field.Type == "object" {
			continue
		}
		r := dal.NewRecord()
		r.Set("name", f.Path)
		r.Set("type", f.Field.Type)
		r.Set("searchable", IsSearchable(f.Field))
		r.Set("filterable", isTrue(f.Field.Filterable))
		r.Set("sortable", isTrue(f.Field.Sortable))
		facetable := false
		switch v := f.Field.Facetable.(type) {
		case bool:
			facetable = v
		case *searchtypes.FacetableOptions:
			facetable = true
		}
		r.Set("facetable", facetable)
		out = append(out, r)
	}
	return out
}

// StableStringify mirrors stableStringify: JSON with object keys sorted
// and undefined members left out.
func StableStringify(value any) string {
	switch t := value.(type) {
	case nil:
		return "null"
	case *dal.Record:
		if t == nil {
			return "null"
		}
		keys := t.Keys()
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			v, _ := t.Get(k)
			kb, _ := dal.MarshalValue(k)
			parts = append(parts, string(kb)+":"+StableStringify(v))
		}
		return "{" + strings.Join(parts, ",") + "}"
	case []any:
		parts := make([]string, len(t))
		for i, v := range t {
			parts[i] = StableStringify(v)
		}
		return "[" + strings.Join(parts, ",") + "]"
	}
	raw, err := dal.MarshalValue(value)
	if err != nil {
		return "null"
	}
	return string(raw)
}

func buildDefinitionHash(fields searchtypes.Fields, settings *searchtypes.Settings, primaryKey, physicalName string) string {
	payload := dal.NewRecord()
	payload.Set("fields", fields.ToRecord())
	payload.Set("settings", settings.ToRecord())
	payload.Set("primary_key", primaryKey)
	payload.Set("physical_name", physicalName)
	sum := sha256.Sum256([]byte(StableStringify(payload)))
	return hex.EncodeToString(sum[:])[:32]
}

func buildPhysicalIndexName(name, prefix string) string {
	var parts []string
	for _, p := range []string{prefix, name} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "_")
}

func get(r *dal.Record, key string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(key)
	return v
}

func copyRecord(r *dal.Record) *dal.Record {
	out := dal.NewRecord()
	if r == nil {
		return out
	}
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		out.Set(k, v)
	}
	return out
}

func stringList(v any) ([]string, bool) {
	arr, ok := v.([]any)
	if !ok {
		if s, ok := v.([]string); ok {
			return s, true
		}
		return nil, false
	}
	out := make([]string, len(arr))
	for i, e := range arr {
		out[i], _ = e.(string)
	}
	return out, true
}

// NormalizeSearchQuery mirrors normalizeSearchQuery: lifts `q` out of the
// filters, applies the pagination defaults and expands shorthand facets.
func NormalizeSearchQuery(query *dal.Record, index *searchtypes.ResolvedDefinition) *searchtypes.ProviderQuery {
	filters, _ := get(query, "filters").(*dal.Record)
	filters = copyRecord(filters)
	var q *string
	if v, ok := filters.Get("q"); ok {
		if s, ok := v.(string); ok {
			q = &s
		}
	}
	filters.Delete("q")

	searchOptions, _ := get(query, "search_options").(*dal.Record)
	searchOptions = copyRecord(searchOptions)
	if facets, ok := get(searchOptions, "facets").([]any); ok {
		expanded := make([]any, len(facets))
		for i, facet := range facets {
			if s, ok := facet.(string); ok {
				r := dal.NewRecord()
				r.Set("field", s)
				r.Set("type", "value")
				expanded[i] = r
				continue
			}
			expanded[i] = facet
		}
		searchOptions.Set("facets", expanded)
	} else if searchOptions.Has("facets") {
		searchOptions.Set("facets", nil)
	}
	if vector, ok := get(searchOptions, "vector").(*dal.Record); ok && get(vector, "field") == nil {
		var vectorFields []FlatField
		for _, f := range FlattenFields(index.Fields, "") {
			if f.Field.Type == "vector" {
				vectorFields = append(vectorFields, f)
			}
		}
		if len(vectorFields) == 1 {
			v := copyRecord(vector)
			v.Set("field", vectorFields[0].Path)
			searchOptions.Set("vector", v)
		}
	}

	attributes := ListRetrievablePaths(index.Fields)
	if fields, ok := stringList(get(query, "fields")); ok && get(query, "fields") != nil {
		attributes = fields
	}
	pagination, _ := get(query, "pagination").(*dal.Record)
	p := dal.NewRecord()
	if v := get(pagination, "skip"); v != nil {
		p.Set("skip", v)
	} else {
		p.Set("skip", float64(0))
	}
	if v := get(pagination, "take"); v != nil {
		p.Set("take", v)
	} else {
		p.Set("take", float64(defaultTake))
	}
	if v := get(pagination, "order"); v != nil {
		p.Set("order", v)
	}
	if v := get(pagination, "cursor"); v != nil {
		p.Set("cursor", v)
	}
	out := &searchtypes.ProviderQuery{
		Index:                index,
		AttributesToRetrieve: attributes,
		Q:                    q,
		Pagination:           p,
		SearchOptions:        searchOptions,
	}
	if filters.Len() > 0 {
		out.Filters = filters
	}
	return out
}

// withoutFieldFilter mirrors withoutFieldFilter.
func withoutFieldFilter(filters *dal.Record, field string) *dal.Record {
	if filters == nil {
		return nil
	}
	next := dal.NewRecord()
	for _, key := range filters.Keys() {
		value, _ := filters.Get(key)
		if key == field {
			continue
		}
		if key == "$and" || key == "$or" {
			if arr, ok := value.([]any); ok {
				var branches []any
				for _, b := range arr {
					br, _ := b.(*dal.Record)
					stripped := withoutFieldFilter(br, field)
					if stripped != nil && stripped.Len() > 0 {
						branches = append(branches, stripped)
					}
				}
				if len(branches) > 0 {
					next.Set(key, branches)
				}
				continue
			}
		}
		if key == "$not" {
			br, _ := value.(*dal.Record)
			stripped := withoutFieldFilter(br, field)
			if stripped != nil && stripped.Len() > 0 {
				next.Set("$not", stripped)
			}
			continue
		}
		next.Set(key, value)
	}
	if next.Len() == 0 {
		return nil
	}
	return next
}

func filtersValue(r *dal.Record) any {
	if r == nil {
		return nil
	}
	return r
}

// BuildDisjunctiveFacetQueries mirrors buildDisjunctiveFacetQueries.
func BuildDisjunctiveFacetQueries(query *searchtypes.ProviderQuery) (*searchtypes.ProviderQuery, []string, map[string]*searchtypes.ProviderQuery) {
	perFacet := map[string]*searchtypes.ProviderQuery{}
	var order []string
	facets, _ := get(query.SearchOptions, "facets").([]any)
	searchOptions := copyRecord(query.SearchOptions)
	searchOptions.Delete("disjunctive_facets")
	base := query.Clone()
	base.SearchOptions = searchOptions
	for _, f := range facets {
		facet, _ := f.(*dal.Record)
		field, _ := get(facet, "field").(string)
		relaxed := withoutFieldFilter(query.Filters, field)
		if StableStringify(filtersValue(relaxed)) == StableStringify(filtersValue(query.Filters)) {
			continue
		}
		pagination := copyRecord(query.Pagination)
		pagination.Set("skip", float64(0))
		pagination.Set("take", float64(0))
		opts := copyRecord(searchOptions)
		opts.Set("facets", []any{facet})
		opts.Set("count", "none")
		fq := base.Clone()
		fq.Filters = relaxed
		fq.Pagination = pagination
		fq.SearchOptions = opts
		if _, ok := perFacet[field]; !ok {
			order = append(order, field)
		}
		perFacet[field] = fq
	}
	return base, order, perFacet
}

// MergeDisjunctiveFacetResults mirrors mergeDisjunctiveFacetResults.
func MergeDisjunctiveFacetResults(base *dal.Record, order []string, perFacet map[string]*dal.Record) *dal.Record {
	if len(order) == 0 {
		return base
	}
	facets := copyRecord(func() *dal.Record { f, _ := get(base, "facets").(*dal.Record); return f }())
	for _, field := range order {
		result := perFacet[field]
		resultFacets, _ := get(result, "facets").(*dal.Record)
		if relaxed := get(resultFacets, field); relaxed != nil {
			facets.Set(field, relaxed)
		}
	}
	out := copyRecord(base)
	out.Set("facets", facets)
	return out
}

func collectFilterPaths(filters *dal.Record, into *[]string, seen map[string]bool) {
	for _, key := range filters.Keys() {
		value, _ := filters.Get(key)
		switch key {
		case "$and", "$or":
			arr, _ := value.([]any)
			for _, b := range arr {
				if br, ok := b.(*dal.Record); ok {
					collectFilterPaths(br, into, seen)
				}
			}
		case "$not":
			if br, ok := value.(*dal.Record); ok {
				collectFilterPaths(br, into, seen)
			}
		default:
			if !seen[key] {
				seen[key] = true
				*into = append(*into, key)
			}
		}
	}
}

func invalid(message string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, message)
}

// ValidateFieldUsage mirrors validateFieldUsage (on a normalized query).
func ValidateFieldUsage(index *searchtypes.ResolvedDefinition, query *searchtypes.ProviderQuery) error {
	fieldMap := buildFieldMap(index.Fields)
	lookup := func(path, usage string) (*searchtypes.FieldDefinition, error) {
		known, ok := fieldMap[path]
		if !ok {
			return nil, invalid(fmt.Sprintf(`Unknown field "%s" used in %s on search index "%s"`, path, usage, index.Name))
		}
		return known.Field, nil
	}
	if query.Filters != nil {
		var paths []string
		collectFilterPaths(query.Filters, &paths, map[string]bool{})
		for _, path := range paths {
			field, err := lookup(path, "filters")
			if err != nil {
				return err
			}
			if !isTrue(field.Filterable) {
				return invalid(fmt.Sprintf(`Field "%s" is not filterable on search index "%s"`, path, index.Name))
			}
		}
	}
	if order, ok := get(query.Pagination, "order").(*dal.Record); ok {
		for _, path := range order.Keys() {
			if path == "_score" {
				continue
			}
			field, err := lookup(path, "pagination.order")
			if err != nil {
				return err
			}
			if !isTrue(field.Sortable) {
				return invalid(fmt.Sprintf(`Field "%s" is not sortable on search index "%s"`, path, index.Name))
			}
		}
	}
	options := query.SearchOptions
	if facets, ok := get(options, "facets").([]any); ok {
		for _, f := range facets {
			var fieldName, kind string
			if s, ok := f.(string); ok {
				fieldName, kind = s, "value"
			} else {
				fr, _ := f.(*dal.Record)
				fieldName, _ = get(fr, "field").(string)
				kind, _ = get(fr, "type").(string)
				if kind == "" {
					kind = "value"
				}
			}
			field, err := lookup(fieldName, "search_options.facets")
			if err != nil {
				return err
			}
			if !IsFacetable(field, kind) {
				return invalid(fmt.Sprintf(`Field "%s" does not support "%s" facets on search index "%s"`, fieldName, kind, index.Name))
			}
		}
	}
	if attrs, ok := stringList(get(options, "attributes_to_search_on")); ok {
		for _, path := range attrs {
			field, err := lookup(path, "search_options.attributes_to_search_on")
			if err != nil {
				return err
			}
			if !IsSearchable(field) {
				return invalid(fmt.Sprintf(`Field "%s" is not searchable on search index "%s"`, path, index.Name))
			}
		}
	}
	if distinct, ok := get(options, "distinct").(string); ok && distinct != "" {
		if _, err := lookup(distinct, "search_options.distinct"); err != nil {
			return err
		}
	}
	if vector, ok := get(options, "vector").(*dal.Record); ok {
		vfield, _ := get(vector, "field").(string)
		if vfield == "" {
			hasVector := false
			for _, f := range fieldMap {
				if f.Field.Type == "vector" {
					hasVector = true
				}
			}
			if hasVector {
				return invalid(`search_options.vector.field is required when the index has more than one vector field`)
			}
			return invalid(fmt.Sprintf(`search_options.vector requires a vector field on search index "%s"`, index.Name))
		}
		field, err := lookup(vfield, "search_options.vector.field")
		if err != nil {
			return err
		}
		if field.Type != "vector" {
			return invalid(fmt.Sprintf(`Field "%s" is not a vector field on search index "%s"`, vfield, index.Name))
		}
		value, _ := get(vector, "value").([]any)
		hasValue := get(vector, "value") != nil
		hasQuery := searchutils.Truthy(get(vector, "query"))
		if hasValue && hasQuery {
			return invalid(`search_options.vector.value and search_options.vector.query are mutually exclusive`)
		}
		if !hasValue && !hasQuery {
			return invalid(`search_options.vector requires either "value" (embedding) or "query" (text to embed)`)
		}
		if hasQuery && !isTrue(field.Embed) {
			return invalid(fmt.Sprintf(`search_options.vector.query requires vector field "%s" to declare "embed"`, vfield))
		}
		if hasValue && field.Dimensions != nil && *field.Dimensions != 0 && len(value) != *field.Dimensions {
			return invalid(fmt.Sprintf(`Vector value for "%s" expected %d dimensions, got %d`, vfield, *field.Dimensions, len(value)))
		}
		if ratio, ok := get(vector, "semantic_ratio").(float64); ok && (ratio < 0 || ratio > 1) {
			return invalid("search_options.vector.semantic_ratio must be between 0 and 1")
		}
	}
	return nil
}

// ResolveIndexDefinitions mirrors resolveIndexDefinitions.
func ResolveIndexDefinitions(definitions []*searchtypes.Definition, defaultProvider, indexPrefix string) (*Indexes, error) {
	resolved := NewIndexes()
	for _, definition := range definitions {
		primaryKey := definition.PrimaryKey
		if primaryKey == "" {
			primaryKey = "id"
		}
		if definition.Name == "" {
			return nil, invalid("A search index definition requires a name")
		}
		if _, ok := resolved.Get(definition.Name); ok {
			return nil, invalid(fmt.Sprintf(`Duplicate search index definition for "%s"`, definition.Name))
		}
		if definition.Fields.Get(primaryKey) == nil {
			return nil, invalid(fmt.Sprintf(`Search index "%s" must declare its primary key field "%s"`, definition.Name, primaryKey))
		}
		physicalName := buildPhysicalIndexName(definition.Name, indexPrefix)
		entry := &searchtypes.ResolvedDefinition{Definition: *definition}
		entry.PrimaryKey = primaryKey
		if entry.Provider == "" {
			entry.Provider = defaultProvider
		}
		if entry.Settings == nil {
			entry.Settings = &searchtypes.Settings{}
		}
		entry.DefinitionHash = buildDefinitionHash(definition.Fields, definition.Settings, primaryKey, physicalName)
		entry.PhysicalName = physicalName
		if err := validateIndexDefinition(entry); err != nil {
			return nil, err
		}
		resolved.set(entry)
	}
	return resolved, nil
}

func validateIndexDefinition(definition *searchtypes.ResolvedDefinition) error {
	fail := func(message string) error {
		return invalid(fmt.Sprintf(`Invalid search index definition "%s": %s`, definition.Name, message))
	}
	hasEvents := len(definition.Events) > 0
	hasConsume := definition.Consume != nil
	if hasEvents && !hasConsume {
		return fail(`declares "events" but no "consume" to turn them into documents`)
	}
	if hasConsume && !hasEvents {
		return fail(`declares "consume" but no "events" for it to be called on`)
	}
	for _, f := range FlattenFields(definition.Fields, "") {
		field, path := f.Field, f.Path
		if field.Type == "vector" && (field.Dimensions == nil || *field.Dimensions == 0) {
			return fail(fmt.Sprintf(`vector field "%s" must declare its "dimensions"`, path))
		}
		if field.Type == "vector" && isTrue(field.Array) {
			return fail(fmt.Sprintf(`vector field "%s" cannot be an array`, path))
		}
		if isTrue(field.Embed) && field.Type != "vector" {
			return fail(fmt.Sprintf(`field "%s" sets "embed", which only applies to vector fields`, path))
		}
		if field.Type == "object" && field.Fields == nil {
			return fail(fmt.Sprintf(`object field "%s" must declare its "fields"`, path))
		}
		numeric := field.Type == "integer" || field.Type == "float" || field.Type == "date"
		if IsFacetable(field, "range") && !numeric {
			return fail(fmt.Sprintf(`field "%s" is not numeric, so it cannot have range facets`, path))
		}
		if IsFacetable(field, "stats") && !numeric {
			return fail(fmt.Sprintf(`field "%s" is not numeric, so it cannot have stats facets`, path))
		}
		if IsSearchable(field) && field.Type != "text" && field.Type != "keyword" {
			return fail(fmt.Sprintf(`field "%s" is of type "%s", which cannot be searched as free text`, path, field.Type))
		}
	}
	return nil
}

// RetrieveIndexDefinition mirrors retrieveIndexDefinition.
func RetrieveIndexDefinition(indexes *Indexes, name string) (*searchtypes.ResolvedDefinition, error) {
	definition, ok := indexes.Get(name)
	if !ok {
		registered := strings.Join(indexes.Names(), ", ")
		if registered == "" {
			registered = "(none)"
		}
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`No search index registered for "%s". Registered indexes: %s`, name, registered))
	}
	return definition, nil
}

// ResolveActiveDefinition mirrors resolveActiveDefinition.
func ResolveActiveDefinition(ctx context.Context, c *Context, name string, fresh bool) (*searchtypes.ResolvedDefinition, error) {
	definition, err := RetrieveIndexDefinition(c.Indexes, name)
	if err != nil {
		return nil, err
	}
	if c.ActiveVersionCache == nil {
		return definition, nil
	}
	active, err := c.ActiveVersionCache.Get(ctx, name, fresh)
	if err != nil {
		return nil, err
	}
	out := *definition
	out.PhysicalName = active.PhysicalName
	out.Provider = active.Provider
	return &out, nil
}

func isNotFound(err error) bool {
	var me *common.MedusaError
	return errors.As(err, &me) && me.Type == common.MedusaErrorTypes.NOT_FOUND
}

// WithActiveIndexRetry mirrors withActiveIndexRetry.
func WithActiveIndexRetry[T any](ctx context.Context, c *Context, run func() (T, error)) (T, error) {
	out, err := run()
	if err == nil || c.ActiveVersionCache == nil || !isNotFound(err) {
		return out, err
	}
	c.ActiveVersionCache.Invalidate()
	return run()
}

// AssertTaskAccepted mirrors assertTaskAccepted.
func AssertTaskAccepted(task searchtypes.Task, index string) (searchtypes.Task, error) {
	if task.Status == "failed" {
		message := "unknown error"
		if task.Error != nil {
			message = task.Error.Error()
		}
		return task, common.NewMedusaError(common.MedusaErrorTypes.UNEXPECTED_STATE, fmt.Sprintf(`Search write failed on index "%s": %s`, index, message))
	}
	return task, nil
}

/* ------------------------------ records ------------------------------ */

// IndexRecord mirrors SearchIndexRecord.
type IndexRecord struct {
	ID            string
	Name          string
	ActiveVersion *int
}

// VersionRecord mirrors SearchIndexVersionRecord.
type VersionRecord struct {
	ID             string
	SearchIndexID  string
	Version        int
	Provider       string
	PhysicalName   string
	DefinitionHash string
	Status         string
}

func toIndexRecord(r *dal.Record) IndexRecord {
	out := IndexRecord{}
	out.ID, _ = get(r, "id").(string)
	out.Name, _ = get(r, "name").(string)
	if v, ok := get(r, "active_version").(float64); ok {
		n := int(v)
		out.ActiveVersion = &n
	}
	return out
}

func toVersionRecord(r *dal.Record) VersionRecord {
	out := VersionRecord{}
	out.ID, _ = get(r, "id").(string)
	out.SearchIndexID, _ = get(r, "search_index_id").(string)
	if v, ok := get(r, "version").(float64); ok {
		out.Version = int(v)
	}
	out.Provider, _ = get(r, "provider").(string)
	out.PhysicalName, _ = get(r, "physical_name").(string)
	out.DefinitionHash, _ = get(r, "definition_hash").(string)
	out.Status, _ = get(r, "status").(string)
	return out
}

func filter(pairs ...any) *dal.Record {
	r := dal.NewRecord()
	for i := 0; i+1 < len(pairs); i += 2 {
		r.Set(pairs[i].(string), pairs[i+1])
	}
	return r
}

func anyStrings(values []string) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

// ListIndexRecords lists SearchIndex rows matching the filters.
func ListIndexRecords(ctx context.Context, c *Context, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]IndexRecord, error) {
	rows, err := c.IndexService.List(ctx, filters, config)
	if err != nil {
		return nil, err
	}
	out := make([]IndexRecord, len(rows))
	for i, r := range rows {
		out[i] = toIndexRecord(r)
	}
	return out, nil
}

// ListVersionRecords lists SearchIndexVersion rows matching the filters.
func ListVersionRecords(ctx context.Context, c *Context, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]VersionRecord, error) {
	rows, err := c.VersionService.List(ctx, filters, config)
	if err != nil {
		return nil, err
	}
	out := make([]VersionRecord, len(rows))
	for i, r := range rows {
		out[i] = toVersionRecord(r)
	}
	return out, nil
}

// ListVersionsByIndexID mirrors listVersionsByIndexId: every version of
// the listed indexes, newest first.
func ListVersionsByIndexID(ctx context.Context, c *Context, indexIDs []string) (map[string][]VersionRecord, error) {
	byID := map[string][]VersionRecord{}
	if len(indexIDs) == 0 {
		return byID, nil
	}
	versions, err := ListVersionRecords(ctx, c, filter("search_index_id", anyStrings(indexIDs)), utilsmodulessdk.FindConfig{Order: []dal.OrderBy{{Field: "version", Direction: "DESC"}}})
	if err != nil {
		return nil, err
	}
	for _, v := range versions {
		byID[v.SearchIndexID] = append(byID[v.SearchIndexID], v)
	}
	return byID, nil
}

// RandomUUID mirrors crypto.randomUUID().
func RandomUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}
