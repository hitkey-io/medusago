// Package confirm mirrors api/admin/products/imports/[transaction_id]/confirm.
package confirm

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// POST mirrors POST: completes the confirmation step.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	key := composer.IdempotencyKey{Action: "invoke", TransactionID: req.Params["transaction_id"], StepID: steps.WaitConfirmationProductImportStepID, WorkflowID: workflows.ImportProductsAsChunksWorkflowID}
	if err := composer.SetStepSuccess(req.Ctx(), req.Scope, key, true); err != nil {
		return err
	}
	return res.Status(202).JSON(dal.NewRecord())
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
