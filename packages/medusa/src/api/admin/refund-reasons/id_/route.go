// Package id_ mirrors api/admin/refund-reasons/[id]/route.ts.
package id_

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/payment-collection/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	r, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "refund_reason", req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if r != nil {
		out.Set("refund_reason", r)
	}
	return res.JSON(out)
}

// Route is the route file.
// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	in := dal.NewRecord()
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		in.Set(k, v)
	}
	in.Set("id", req.Params["id"])
	if _, err := workflows.UpdateRefundReasonsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{in}); err != nil {
		return err
	}
	return GET(req, res)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeleteRefundReasonsWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "refund_reason")
	out.Set("deleted", true)
	return res.JSON(out)
}

var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
