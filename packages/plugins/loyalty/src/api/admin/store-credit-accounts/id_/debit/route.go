// Package debit mirrors api/admin/store-credit-accounts/[id]/debit of
// @medusajs/loyalty-plugin.
package debit

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	storecreditaccounts "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/admin/store-credit-accounts"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/admin/store-credit-accounts/id_/credit"
	scworkflows "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/store-credit/workflows"
)

// POST mirrors POST /admin/store-credit-accounts/:id/debit.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if _, err := scworkflows.DebitStoreCreditAccountWorkflow.Run(req.Ctx(), req.Scope, credit.TransactionInput(req)); err != nil {
		return err
	}
	f := dal.NewRecord()
	f.Set("id", req.Params["id"])
	account, err := storecreditaccounts.RetrieveAccount(req, "store_credit_account", f)
	if err != nil {
		return err
	}
	return storecreditaccounts.AccountResponse(res, account)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
