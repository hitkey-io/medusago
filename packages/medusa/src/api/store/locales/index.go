// Package locales mirrors api/store/locales (route.ts, middlewares.ts).
package locales

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Middlewares mirrors storeLocalesRoutesMiddlewares (no middleware).
var Middlewares = []medusahttp.MiddlewareRoute{}

// GET mirrors GET: the locales supported by the store.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	pagination := dal.NewRecord()
	pagination.Set("take", float64(1))
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: "store", Fields: []string{"supported_locales.*", "supported_locales.locale.*"}, Pagination: pagination}, joiner.Options{})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if len(r.Data) > 0 {
		store, _ := r.Data[0].(*dal.Record)
		var supported []any
		if store != nil {
			v, _ := store.Get("supported_locales")
			supported, _ = v.([]any)
		}
		locales := []any{}
		for _, x := range supported {
			sl, ok := x.(*dal.Record)
			if !ok {
				continue
			}
			l := dal.NewRecord()
			code, _ := sl.Get("locale_code")
			l.Set("code", code)
			var name any
			if loc, ok := sl.Get("locale"); ok {
				if lr, ok := loc.(*dal.Record); ok {
					name, _ = lr.Get("name")
				}
			}
			l.Set("name", name)
			locales = append(locales, l)
		}
		out.Set("locales", locales)
	}
	return res.JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
