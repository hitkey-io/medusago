// Package query mirrors @medusajs/query/src/query: Query.graph and its
// conversion to RemoteJoiner queries.
package query

import (
	"strings"

	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	typesjoiner "github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

const (
	fieldsKey = "__fields"
	argsKey   = "__args"
)

// GraphInput mirrors RemoteQueryInput.
type GraphInput struct {
	Entity      string
	Fields      []string
	Filters     *dal.Record
	Pagination  *dal.Record
	WithDeleted bool
	// Context mirrors the query context: QueryContext records (see
	// QueryContext) at the entity level or under relation keys.
	Context *dal.Record
}

const queryContextType = "QueryContext"

// QueryContext mirrors QueryContext(value): a record marked as a query
// context.
func QueryContext(value *dal.Record) *dal.Record {
	out := dal.NewRecord()
	out.Set("__type", queryContextType)
	if value != nil {
		for _, k := range value.Keys() {
			v, _ := value.Get(k)
			out.Set(k, v)
		}
	}
	return out
}

func isQueryContext(v any) bool {
	r, ok := v.(*dal.Record)
	if !ok {
		return false
	}
	t, _ := r.Get("__type")
	return t == queryContextType
}

// processNestedContext mirrors processNestedObjects of toRemoteQuery.
func processNestedContext(target *dal.Record, source *dal.Record, topLevel bool, withDeleted bool) {
	for _, key := range source.Keys() {
		var src any = source
		if !topLevel {
			src, _ = source.Get(key)
		}
		rec, isObj := src.(*dal.Record)
		if !isObj {
			target.Set(key, src)
			continue
		}
		if isQueryContext(rec) {
			normalized := dal.NewRecord()
			for _, k := range rec.Keys() {
				if k == "__type" {
					continue
				}
				v, _ := rec.Get(k)
				normalized.Set(k, v)
			}
			t := target
			if !topLevel {
				t = child(target, key)
			}
			args := child(t, argsKey)
			args.Set("context", normalized)
			if withDeleted {
				args.Set("withDeleted", true)
			}
			continue
		}
		next := target
		if !topLevel {
			next = child(target, key)
		}
		processNestedContext(next, rec, false, withDeleted)
	}
}

func child(r *dal.Record, key string) *dal.Record {
	if v, ok := r.Get(key); ok {
		if c, ok := v.(*dal.Record); ok {
			return c
		}
	}
	c := dal.NewRecord()
	r.Set(key, c)
	return c
}

func appendField(r *dal.Record, field string) {
	v, _ := r.Get(fieldsKey)
	list, _ := v.([]string)
	r.Set(fieldsKey, append(list, field))
}

// ToRemoteQuery mirrors toRemoteQuery.
func ToRemoteQuery(in GraphInput, joinerConfigs []*typesjoiner.ModuleJoinerConfig) *dal.Record {
	query := dal.NewRecord()
	entry := dal.NewRecord()
	entry.Set(fieldsKey, []string{})
	query.Set(in.Entity, entry)
	if in.Context != nil {
		processNestedContext(entry, in.Context, true, in.WithDeleted)
	}

	for _, field := range in.Fields {
		if !strings.Contains(field, ".") {
			appendField(entry, field)
			continue
		}
		segments := strings.Split(field, ".")
		prop := segments[len(segments)-1]
		ref := entry
		for _, seg := range segments[:len(segments)-1] {
			ref = child(ref, seg)
		}
		appendField(ref, prop)
	}

	if in.Pagination != nil {
		args := child(entry, argsKey)
		for _, k := range in.Pagination.Keys() {
			v, _ := in.Pagination.Get(k)
			args.Set(k, v)
		}
	}
	if in.WithDeleted {
		child(entry, argsKey).Set("withDeleted", true)
	}
	if in.Filters != nil {
		parseAndAssignFilters(in.Entity, in.Filters, query, false, joinerConfigs)
	}
	return query
}

type lookup struct {
	config *typesjoiner.ModuleJoinerConfig
}

func findAlias(cfg *typesjoiner.ModuleJoinerConfig, entryPoint string) bool {
	for _, a := range cfg.Alias {
		for _, n := range a.Name {
			if n == entryPoint {
				return true
			}
		}
	}
	return false
}

func retrieveJoinerConfig(entryPoint string, parent *typesjoiner.ModuleJoinerConfig, configs []*typesjoiner.ModuleJoinerConfig) *typesjoiner.ModuleJoinerConfig {
	if parent != nil && findAlias(parent, entryPoint) {
		return parent
	}
	for _, c := range configs {
		if findAlias(c, entryPoint) {
			return c
		}
	}
	return nil
}

// parseAndAssignFilters mirrors parseAndAssignFilters: filters on another
// module's alias are moved to that expand.
func parseAndAssignFilters(entryPoint string, filters *dal.Record, remote *dal.Record, isFieldAliasNestedRelation bool, configs []*typesjoiner.ModuleJoinerConfig) {
	entryConfig := retrieveJoinerConfig(entryPoint, nil, configs)
	for _, filterKey := range filters.Keys() {
		filterValue, _ := filters.Get(filterKey)
		obj, isObj := filterValue.(*dal.Record)
		if isObj && !isOperatorRecord(obj) {
			for _, nestedKey := range obj.Keys() {
				nestedValue, _ := obj.Get(nestedKey)
				keyConfig := retrieveJoinerConfig(nestedKey, entryConfig, configs)
				if keyConfig == nil || (entryConfig != nil && keyConfig.ServiceName == entryConfig.ServiceName) {
					assignNested(entryPoint, filterKey, nestedKey, nestedValue, remote, isFieldAliasNestedRelation)
					continue
				}
				nestedIsAlias := isFieldAliasNestedRelationHelper(nestedKey, entryConfig, keyConfig, configs)
				target := child(child(remote, entryPoint), filterKey)
				if nestedFilters, ok := nestedValue.(*dal.Record); ok {
					parseAndAssignFilters(nestedKey, nestedFilters, target, nestedIsAlias, configs)
				}
			}
			continue
		}
		assign(entryPoint, filterKey, filterValue, remote, isFieldAliasNestedRelation)
	}
}

func isOperatorRecord(r *dal.Record) bool {
	if r.Len() == 0 {
		return false
	}
	for _, k := range r.Keys() {
		if !strings.HasPrefix(k, "$") {
			return false
		}
	}
	return true
}

func filtersOf(remote *dal.Record, entryPoint string) *dal.Record {
	return child(child(child(remote, entryPoint), argsKey), "filters")
}

func assign(entryPoint, key string, value any, remote *dal.Record, nested bool) {
	f := filtersOf(remote, entryPoint)
	if nested {
		f = child(f, entryPoint)
	}
	f.Set(key, value)
}

func assignNested(entryPoint, key, nestedKey string, nestedValue any, remote *dal.Record, nested bool) {
	f := filtersOf(remote, entryPoint)
	if nested {
		f = child(f, entryPoint)
	}
	child(f, key).Set(nestedKey, nestedValue)
}

func isFieldAliasNestedRelationHelper(nestedKey string, entry, keyConfig *typesjoiner.ModuleJoinerConfig, configs []*typesjoiner.ModuleJoinerConfig) bool {
	if entry == nil {
		return false
	}
	for _, c := range configs {
		if len(c.Relationships) == 0 {
			continue
		}
		all := true
		for _, rel := range c.Relationships {
			if rel.ServiceName != entry.ServiceName && rel.ServiceName != keyConfig.ServiceName {
				all = false
				break
			}
		}
		if !all {
			continue
		}
		for _, ext := range c.Extends {
			if _, ok := ext.FieldAlias[nestedKey]; ok {
				return true
			}
		}
		return false
	}
	return false
}

// ToRemoteJoinerQuery mirrors toRemoteJoinerQuery: the nested query object
// becomes the root fields/args plus a depth-first list of expands.
func ToRemoteJoinerQuery(obj *dal.Record) *joiner.RemoteJoinerQuery {
	q := &joiner.RemoteJoinerQuery{Fields: []string{}}
	var walk func(r *dal.Record, parent string, isEntry bool)
	walk = func(r *dal.Record, parent string, isEntry bool) {
		for _, key := range r.Keys() {
			if key == fieldsKey || key == argsKey || key == "__directives" || key == "fields" {
				continue
			}
			v, _ := r.Get(key)
			value, ok := v.(*dal.Record)
			if !ok {
				continue
			}
			name := key
			if parent != "" {
				name = parent + "." + key
			}
			var fields []string
			if f, ok := value.Get(fieldsKey); ok {
				fields, _ = f.([]string)
			}
			var args []joiner.Arg
			if a, ok := value.Get(argsKey); ok {
				if ar, ok := a.(*dal.Record); ok {
					for _, k := range ar.Keys() {
						val, _ := ar.Get(k)
						args = append(args, joiner.Arg{Name: k, Value: val})
					}
				}
			}
			if isEntry {
				q.Alias = key
				q.Args = args
				if fields != nil {
					q.Fields = fields
				}
				walk(value, "", false)
				continue
			}
			q.Expands = append(q.Expands, joiner.ExpandSpec{Property: name, Fields: fields, Args: args})
			walk(value, name, false)
		}
	}
	walk(obj, "", true)
	return q
}
