package query

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// ModuleService is what Query reads from: a module service (list<suffix>)
// or a link module service (list).
type ModuleService interface {
	List(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error)
	ListAndCount(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, int, error)
}

// ModuleDataFetcher mirrors ModuleDataFetcher.
type ModuleDataFetcher struct {
	Services map[string]ModuleService
}

// fieldsAndRelations mirrors getAllFieldsAndRelations.
func fieldsAndRelations(expand *joiner.ExpandProperty, prefix string, args map[string]any, argOrder *[]string) (select_ []string, selectAll bool, relations []string) {
	var fields []string
	seen := map[string]bool{}
	add := func(f string) {
		if !seen[f] {
			seen[f] = true
			fields = append(fields, f)
		}
	}
	remove := func(f string) {
		if seen[f] {
			delete(seen, f)
			for i, x := range fields {
				if x == f {
					fields = append(fields[:i], fields[i+1:]...)
					break
				}
			}
		}
	}
	for _, f := range expand.Fields {
		if f == "*" {
			selectAll = true
			break
		}
		if prefix != "" {
			add(prefix + "." + f)
		} else {
			add(f)
		}
	}
	for _, a := range expand.Args {
		if a.Name == "filters" && a.Value != nil {
			if _, ok := args[prefix]; !ok {
				*argOrder = append(*argOrder, prefix)
			}
			args[prefix] = a.Value
		}
	}
	if expand.Expands != nil {
		for _, key := range expand.Expands.Keys {
			newPrefix := key
			if prefix != "" {
				newPrefix = prefix + "." + key
			}
			relations = append(relations, newPrefix)
			remove(newPrefix)
			sub, subAll, subRelations := fieldsAndRelations(expand.Expands.Values[key], newPrefix, args, argOrder)
			// result.select is undefined when the relation selects "*"
			if !subAll {
				for _, f := range sub {
					add(f)
				}
			}
			relations = append(relations, subRelations...)
		}
	}
	return fields, selectAll, relations
}

func toPascalCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// Fetch mirrors executeFetchRequest.
func (f *ModuleDataFetcher) Fetch(ctx context.Context, expand *joiner.ExpandProperty, keyField string, ids []any, hasIDs bool, relationship *joiner.Relationship) (*joiner.FetchResult, error) {
	service, ok := f.Services[expand.ServiceConfig.ServiceName]
	if !ok {
		return nil, fmt.Errorf("module %s is not loaded", expand.ServiceConfig.ServiceName)
	}

	args := map[string]any{}
	var argOrder []string
	fields, selectAll, relations := fieldsAndRelations(expand, "", args, &argOrder)

	filters := dal.NewRecord()
	config := utilsmodulessdk.FindConfig{Relations: relations}
	if !selectAll {
		config.Select = fields
		if config.Select == nil {
			config.Select = []string{}
		}
	}
	hasPagination := false
	for _, a := range expand.Args {
		switch a.Name {
		case "filters":
			if rec, ok := a.Value.(*dal.Record); ok {
				for _, k := range rec.Keys() {
					v, _ := rec.Get(k)
					filters.Set(k, v)
				}
			}
		case "skip", "offset":
			if n, ok := toIntPtr(a.Value); ok {
				config.Skip = n
			}
			hasPagination = true
		case "cursor":
			hasPagination = true
		case "take", "limit":
			config.Take, _ = toIntPtr(a.Value)
		case "order", "sort":
			if rec, ok := a.Value.(*dal.Record); ok {
				for _, k := range rec.Keys() {
					dir, _ := rec.Get(k)
					config.Order = append(config.Order, dal.OrderBy{Field: k, Direction: strings.ToUpper(fmt.Sprint(dir))})
				}
			}
		case "withDeleted":
			config.WithDeleted, _ = a.Value.(bool)
		case "context":
			if a.Value != nil {
				filters.Set("context", a.Value)
			}
		}
	}
	// Filters of nested relations (args other than the root prefix).
	for _, prefix := range argOrder {
		if prefix == "" {
			continue
		}
		filters.Set(prefix, args[prefix])
	}

	if hasIDs {
		if len(ids) == 0 {
			if hasPagination {
				meta := paginationMetadata(config, 0)
				return &joiner.FetchResult{Rows: []any{}, Metadata: meta, Paged: true}, nil
			}
			return &joiner.FetchResult{Rows: []any{}}, nil
		}
		filters.Set(keyField, ids)
		if !hasPagination {
			config.Take = nil
		}
	}

	suffix := ""
	if relationship != nil {
		if s, ok := relationship.Args["methodSuffix"].(string); ok {
			suffix = toPascalCase(s)
		}
	}
	if suffix == "" {
		if s, ok := expand.ServiceConfig.Args["methodSuffix"].(string); ok {
			suffix = toPascalCase(s)
		}
	}

	if hasPagination {
		rows, count, err := service.ListAndCount(ctx, suffix, filters, config)
		if err != nil {
			return nil, err
		}
		return &joiner.FetchResult{Rows: normalizeRows(rows), Metadata: paginationMetadata(config, count), Paged: true}, nil
	}
	rows, err := service.List(ctx, suffix, filters, config)
	if err != nil {
		return nil, err
	}
	return &joiner.FetchResult{Rows: normalizeRows(rows)}, nil
}

func toIntPtr(v any) (*int, bool) {
	switch t := v.(type) {
	case float64:
		n := int(t)
		return &n, true
	case int:
		return &t, true
	}
	return nil, false
}

func paginationMetadata(config utilsmodulessdk.FindConfig, count int) *dal.Record {
	meta := dal.NewRecord()
	if config.Skip != nil {
		meta.Set("skip", float64(*config.Skip))
	} else {
		meta.Set("skip", nil)
	}
	if config.Take != nil {
		meta.Set("take", float64(*config.Take))
	} else {
		meta.Set("take", nil)
	}
	meta.Set("count", float64(count))
	return meta
}

// normalizeRows converts records so that collections are []any, the
// representation the joiner works with.
func normalizeRows(rows []*dal.Record) []any {
	out := make([]any, len(rows))
	for i, r := range rows {
		out[i] = normalizeValue(r)
	}
	return out
}

func normalizeValue(v any) any {
	switch t := v.(type) {
	case []*dal.Record:
		out := make([]any, len(t))
		for i, r := range t {
			out[i] = normalizeValue(r)
		}
		return out
	case *dal.Record:
		if t == nil {
			return nil
		}
		for _, k := range t.Keys() {
			x, _ := t.Get(k)
			t.Set(k, normalizeValue(x))
		}
		return t
	default:
		return v
	}
}
