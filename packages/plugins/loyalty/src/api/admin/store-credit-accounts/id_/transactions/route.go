// Package transactions mirrors
// api/admin/store-credit-accounts/[id]/transactions of
// @medusajs/loyalty-plugin.
package transactions

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET /admin/store-credit-accounts/:id/transactions.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	f := dal.NewRecord()
	f.Set("account_id", req.Params["id"])
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			f.Set(k, v)
		}
	}
	r, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{
		Entity: "account_transaction", IDOrFilter: f, Fields: req.QueryConfig.Fields, Pagination: req.QueryConfig.Pagination,
	})
	if err != nil {
		return err
	}
	return res.JSON(httputils.ListResponse("transactions", r))
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
