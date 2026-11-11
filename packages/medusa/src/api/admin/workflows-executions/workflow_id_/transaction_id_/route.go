// Package transaction_id_ mirrors api/admin/workflows-executions/[workflow_id]/[transaction_id].
package transaction_id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET: remoteQuery({ entryPoint: "workflow_execution",
// variables: { workflow_id, transaction_id } })[0]. Both variables are
// primary keys of the workflows module: the joiner keeps the first one as
// the primary key argument and drops the others, so the executions are
// filtered by workflow_id only and the first one (by id) is returned,
// whatever the transaction id.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	vars := dal.NewRecord()
	vars.Set("workflow_id", req.Params["workflow_id"])
	vars.Set("transaction_id", req.Params["transaction_id"])
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "workflow_execution", vars, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if len(r.Data) > 0 && r.Data[0] != nil {
		out.Set("workflow_execution", r.Data[0])
	}
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
