package utils

import (
	"strings"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// CrossModuleFilterOptions mirrors the options of applyCrossModuleFilter.
type CrossModuleFilterOptions struct {
	FilterableField string
	RelationPath    string
}

// isPresent mirrors isPresent.
func isPresent(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case []any:
		return len(t) > 0
	case *dal.Record:
		return t != nil && t.Len() > 0
	}
	return true
}

// ApplyCrossModuleFilter mirrors applyCrossModuleFilter: it moves a flat
// query param filtering by a linked module's field (e.g. sales_channel_id)
// into a nested Query filter along RelationPath (e.g.
// { sales_channels: { id } }), applied by query.graph through cross-module
// joins.
func ApplyCrossModuleFilter(o CrossModuleFilterOptions) medusahttp.MiddlewareFunction {
	return func(req *medusahttp.MedusaRequest, _ *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		filters := req.FilterableFields
		if filters == nil {
			next(nil)
			return nil
		}
		value, _ := filters.Get(o.FilterableField)
		if !isPresent(value) {
			next(nil)
			return nil
		}
		filters.Delete(o.FilterableField)

		segments := strings.Split(o.RelationPath, ".")
		field := segments[len(segments)-1]
		node := filters
		for _, segment := range segments[:len(segments)-1] {
			current, _ := node.Get(segment)
			child, ok := current.(*dal.Record)
			if !ok || child == nil {
				child = dal.NewRecord()
				node.Set(segment, child)
			}
			node = child
		}
		node.Set(field, value)
		next(nil)
		return nil
	}
}
