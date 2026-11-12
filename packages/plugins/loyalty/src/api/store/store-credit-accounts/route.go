package storecreditaccounts

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	admin "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/admin/store-credit-accounts"
)

// GET mirrors GET /store/store-credit-accounts: the accounts of the
// authenticated customer.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	f := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			f.Set(k, v)
		}
	}
	f.Set("customer_id", req.AuthString("actor_id"))
	r, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{
		Entity: "store_credit_account", IDOrFilter: f, Fields: req.QueryConfig.Fields, Pagination: req.QueryConfig.Pagination,
	})
	if err != nil {
		return err
	}
	accounts, err := admin.WithAccountStats(req, r.Data)
	if err != nil {
		return err
	}
	r.Data = accounts
	return res.JSON(httputils.ListResponse("store_credit_accounts", r))
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
