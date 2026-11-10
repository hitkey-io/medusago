// Package run mirrors api/admin/workflows-executions/[workflow_id]/run.
package run

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// POST mirrors POST: workflowEngine.run(workflow_id, { transactionId, input }).
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	input, _ := body.Get("input")
	transactionID, _ := body.Get("transaction_id")
	tid, _ := transactionID.(string)
	workflowID := req.Params["workflow_id"]
	_, info, err := composer.RunByID(req.Ctx(), req.Scope, workflowID, input, composer.RunOptions{TransactionID: tid})
	if err != nil {
		return err
	}
	ack := dal.NewRecord()
	ack.Set("transactionId", info.TransactionID)
	ack.Set("workflowId", workflowID)
	ack.Set("hasFinished", !info.Detached)
	ack.Set("hasFailed", info.State == composer.TransactionStateReverted || info.State == composer.TransactionStateFailed)
	out := dal.NewRecord()
	out.Set("acknowledgement", ack)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
