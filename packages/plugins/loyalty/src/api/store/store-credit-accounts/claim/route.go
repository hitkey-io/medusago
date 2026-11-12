// Package claim mirrors api/store/store-credit-accounts/claim of
// @medusajs/loyalty-plugin.
package claim

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	admin "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/admin/store-credit-accounts"
	scworkflows "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/store-credit/workflows"
)

// POST mirrors POST /store/store-credit-accounts/claim (the raw body code).
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	var code string
	if body, ok := req.Body.(*dal.Record); ok && body != nil {
		v, _ := body.Get("code")
		code, _ = v.(string)
	}
	if _, err := scworkflows.ClaimStoreCreditAccountWorkflow.Run(req.Ctx(), req.Scope, scworkflows.ClaimStoreCreditAccountWorkflowInput{
		Code: code, CustomerID: req.AuthString("actor_id"),
	}); err != nil {
		return err
	}
	f := dal.NewRecord()
	f.Set("code", code)
	r, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{
		Entity: "store_credit_account", IDOrFilter: f, Fields: []string{"id", "code", "customer_id", "currency_code", "balance"},
	})
	if err != nil {
		return err
	}
	var account any
	if len(r.Data) > 0 {
		account = r.Data[0]
	}
	return admin.AccountResponse(res, account)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
