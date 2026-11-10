package products

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// MaybeApplyPriceListsFilter mirrors maybeApplyPriceListsFilter.
func MaybeApplyPriceListsFilter() medusahttp.MiddlewareFunction {
	return func(req *medusahttp.MedusaRequest, _ *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		ff := req.FilterableFields
		ids, _ := ff.Get("price_list_id")
		if ids == nil {
			next(nil)
			return nil
		}
		// With the index engine (and no tags/categories filter forcing the
		// regular list), the route filters on variants.prices.price_list_id.
		if indexEngineEnabled(req) && !truthy(filterValue(ff, "tags")) && !truthy(filterValue(ff, "categories")) {
			next(nil)
			return nil
		}
		ff.Delete("price_list_id")
		f := dal.NewRecord()
		f.Set("price_list_id", ids)
		vars := dal.NewRecord()
		vars.Set("filters", f)
		r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "price", vars, []string{"price_set.variant.id"})
		if err != nil {
			return err
		}
		seen := map[string]bool{}
		var variantIDs []any
		for _, row := range r.Data {
			p, _ := row.(*dal.Record)
			ps, _ := p.Get("price_set")
			psr, _ := ps.(*dal.Record)
			if psr == nil {
				continue
			}
			v, _ := psr.Get("variant")
			vr, _ := v.(*dal.Record)
			if vr == nil {
				continue
			}
			id, _ := vr.Get("id")
			if s, ok := id.(string); ok && s != "" && !seen[s] {
				seen[s] = true
				variantIDs = append(variantIDs, s)
			}
		}
		variants := dal.NewRecord()
		if cur, ok := ff.Get("variants"); ok {
			if r, ok := cur.(*dal.Record); ok {
				for _, k := range r.Keys() {
					x, _ := r.Get(k)
					variants.Set(k, x)
				}
			}
		}
		if variantIDs == nil {
			variantIDs = []any{}
		}
		variants.Set("id", variantIDs)
		ff.Set("variants", variants)
		next(nil)
		return nil
	}
}

// truthy mirrors JavaScript truthiness for filter values.
func truthy(v any) bool {
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
