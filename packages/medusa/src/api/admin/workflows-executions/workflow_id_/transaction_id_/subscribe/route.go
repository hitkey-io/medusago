// Package subscribe mirrors api/admin/workflows-executions/[workflow_id]/[transaction_id]/subscribe.
package subscribe

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	workflowsubscribe "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/workflows-executions/workflow_id_/subscribe"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return workflowsubscribe.Stream(req, res, req.Params["workflow_id"], req.Params["transaction_id"])
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
