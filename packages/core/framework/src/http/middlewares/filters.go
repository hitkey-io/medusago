package middlewares

import (
	"sort"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func ensureFilterable(req *medusahttp.MedusaRequest) *dal.Record {
	if req.FilterableFields == nil {
		req.FilterableFields = dal.NewRecord()
	}
	return req.FilterableFields
}

// ApplyParamsAsFilters mirrors applyParamsAsFilters.
func ApplyParamsAsFilters(mappings map[string]string) medusahttp.MiddlewareFunction {
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		keys := make([]string, 0, len(req.Params))
		for k := range req.Params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, param := range keys {
			if to := mappings[param]; to != "" {
				ensureFilterable(req).Set(to, req.Params[param])
			}
		}
		next(nil)
		return nil
	}
}

// ClearFiltersByKey mirrors clearFiltersByKey.
func ClearFiltersByKey(keys ...string) medusahttp.MiddlewareFunction {
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		for _, k := range keys {
			ensureFilterable(req).Delete(k)
		}
		next(nil)
		return nil
	}
}

// DefaultFilter is one applyDefaultFilters entry: Value, or Fn computing it
// from (and possibly mutating) the filterable fields.
type DefaultFilter struct {
	Key   string
	Value any
	Fn    func(filters *dal.Record, fields []string) any
}

func filterIsPresent(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case []any:
		return len(t) > 0
	case []string:
		return len(t) > 0
	case *dal.Record:
		return t.Len() > 0
	}
	return true
}

// ApplyDefaultFilters mirrors applyDefaultFilters.
func ApplyDefaultFilters(filters ...DefaultFilter) medusahttp.MiddlewareFunction {
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		ff := ensureFilterable(req)
		for _, f := range filters {
			v := f.Value
			if f.Fn != nil {
				v = f.Fn(ff, req.QueryConfig.Fields)
			}
			if obj, ok := v.(*dal.Record); ok {
				merged := dal.NewRecord()
				if existing, ok := ff.Get(f.Key); ok {
					if er, ok := existing.(*dal.Record); ok {
						for _, k := range er.Keys() {
							x, _ := er.Get(k)
							merged.Set(k, x)
						}
					}
				}
				for _, k := range obj.Keys() {
					x, _ := obj.Get(k)
					merged.Set(k, x)
				}
				ff.Set(f.Key, merged)
			} else if filterIsPresent(v) {
				ff.Set(f.Key, v)
			}
		}
		next(nil)
		return nil
	}
}
