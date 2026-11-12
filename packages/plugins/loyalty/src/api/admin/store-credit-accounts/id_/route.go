// Package id_ mirrors api/admin/store-credit-accounts/[id] of
// @medusajs/loyalty-plugin.
package id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	storecreditaccounts "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/admin/store-credit-accounts"
)

// GET mirrors GET /admin/store-credit-accounts/:id.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	f := dal.NewRecord()
	f.Set("id", req.Params["id"])
	account, err := storecreditaccounts.RetrieveAccount(req, "store_credit_account", f)
	if err != nil {
		return err
	}
	return storecreditaccounts.AccountResponse(res, account)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
