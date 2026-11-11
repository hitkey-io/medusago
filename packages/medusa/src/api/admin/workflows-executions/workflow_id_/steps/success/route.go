// Package success mirrors api/admin/workflows-executions/[workflow_id]/steps/success.
package success

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	get := func(k string) any {
		v, _ := body.Get(k)
		return v
	}
	action := "invoke"
	if a, ok := get("action").(string); ok && a != "" {
		action = a
	}
	key := composer.IdempotencyKey{Action: action, TransactionID: fmt.Sprint(get("transaction_id")), StepID: fmt.Sprint(get("step_id")), WorkflowID: req.Params["workflow_id"]}
	if err := composer.SetStepSuccess(req.Ctx(), req.Scope, key, get("response"), composer.StepResponseOptions{WithContainer: true}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("success", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
