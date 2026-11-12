// Package credit mirrors api/admin/store-credit-accounts/[id]/credit of
// @medusajs/loyalty-plugin.
package credit

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	storecreditaccounts "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/admin/store-credit-accounts"
	scworkflows "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/store-credit/workflows"
)

// TransactionInput builds the credit / debit workflow input from the
// validated body and the authenticated actor.
func TransactionInput(req *medusahttp.MedusaRequest) scworkflows.StoreCreditAccountTransactionInput {
	body, _ := req.ValidatedBody.(*dal.Record)
	in := scworkflows.StoreCreditAccountTransactionInput{
		AccountID:   req.Params["id"],
		Reference:   req.AuthString("actor_type"),
		ReferenceID: req.AuthString("actor_id"),
	}
	if body != nil {
		if v, ok := body.Get("amount"); ok {
			in.Amount, _ = v.(float64)
		}
		if v, ok := body.Get("note"); ok {
			in.Note, _ = v.(string)
		}
	}
	return in
}

// POST mirrors POST /admin/store-credit-accounts/:id/credit.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if _, err := scworkflows.CreditStoreCreditAccountWorkflow.Run(req.Ctx(), req.Scope, TransactionInput(req)); err != nil {
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
