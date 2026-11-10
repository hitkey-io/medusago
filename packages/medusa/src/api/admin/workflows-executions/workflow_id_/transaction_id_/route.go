// Package transaction_id_ mirrors api/admin/workflows-executions/[workflow_id]/[transaction_id].
package transaction_id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET (undefined when missing).
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	f := dal.NewRecord()
	f.Set("workflow_id", req.Params["workflow_id"])
	f.Set("transaction_id", req.Params["transaction_id"])
	v, err := httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "workflow_execution", f, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if v != nil {
		out.Set("workflow_execution", v)
	}
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
