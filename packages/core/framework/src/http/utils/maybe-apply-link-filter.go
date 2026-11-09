package utils

import (
	"strings"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// LinkFilterOptions mirrors the options of maybeApplyLinkFilter.
type LinkFilterOptions struct {
	EntryPoint      string
	ResourceID      string
	FilterableField string
	// FilterByField defaults to "id".
	FilterByField string
}

func truthyValue(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case bool:
		return t
	case float64:
		return t != 0
	}
	return true
}

// MaybeApplyLinkFilter mirrors maybeApplyLinkFilter.
func MaybeApplyLinkFilter(o LinkFilterOptions) medusahttp.MiddlewareFunction {
	if o.FilterByField == "" {
		o.FilterByField = "id"
	}
	return func(req *medusahttp.MedusaRequest, _ *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		ff := req.FilterableFields
		if ff == nil {
			next(nil)
			return nil
		}
		v, _ := ff.Get(o.FilterableField)
		if !truthyValue(v) {
			next(nil)
			return nil
		}
		ids, ok := v.([]any)
		if !ok {
			ids = []any{v}
		}
		ff.Delete(o.FilterableField)
		existing, _ := ff.Get(o.FilterByField)
		filters := dal.NewRecord()
		filters.Set(o.FilterableField, ids)
		if truthyValue(existing) {
			filters.Set(o.ResourceID, existing)
		}
		res, err := QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: o.EntryPoint, Fields: []string{o.ResourceID}, Filters: filters}, joiner.Options{})
		if err != nil {
			return err
		}
		out := make([]any, 0, len(res.Data))
		for _, row := range res.Data {
			if r, ok := row.(*dal.Record); ok {
				x, _ := r.Get(o.ResourceID)
				out = append(out, x)
			} else {
				out = append(out, nil)
			}
		}
		ff.Set(o.FilterByField, out)
		req.FilterableFields = transformFilterableFields(ff)
		next(nil)
		return nil
	}
}

// transformFilterableFields mirrors transformFilterableFields.
func transformFilterableFields(ff *dal.Record) *dal.Record {
	result := dal.NewRecord()
	for _, key := range ff.Keys() {
		value, _ := ff.Get(key)
		parts := strings.Split(key, ".")
		cur := result
		for i, part := range parts {
			if i == len(parts)-1 {
				cur.Set(part, value)
				break
			}
			next, ok := cur.Get(part)
			r, isRec := next.(*dal.Record)
			if !ok || !isRec {
				r = dal.NewRecord()
				cur.Set(part, r)
			}
			cur = r
		}
	}
	return result
}
