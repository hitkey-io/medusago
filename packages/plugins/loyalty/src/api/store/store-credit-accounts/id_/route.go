// Package id_ mirrors api/store/store-credit-accounts/[id] of
// @medusajs/loyalty-plugin.
package id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	admin "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/admin/store-credit-accounts"
)

// GET mirrors GET /store/store-credit-accounts/:id.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	f := dal.NewRecord()
	f.Set("id", id)
	f.Set("customer_id", req.AuthString("actor_id"))
	account, err := admin.RetrieveAccount(req, "store_credit_account", f)
	if err != nil {
		return err
	}
	rec, _ := account.(*dal.Record)
	merged, err := admin.MergeAccountStats(req, rec, id)
	if err != nil {
		return err
	}
	return admin.AccountResponse(res, merged)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
