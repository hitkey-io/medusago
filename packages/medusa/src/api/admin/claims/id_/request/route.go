// Package request mirrors api/admin/claims/[id]/request.
package request

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/claims"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/claims/:id/request.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	preview, err := workflows.ConfirmClaimRequestWorkflow.Run(req.Ctx(), req.Scope, workflows.ConfirmClaimRequestInput{ClaimID: req.Params["id"], ConfirmedBy: orders.ActorID(req)})
	if err != nil {
		return err
	}
	v := dal.NewRecord()
	v.Set("id", req.Params["id"])
	f := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			x, _ := req.FilterableFields.Get(k)
			f.Set(k, x)
		}
	}
	v.Set("filters", f)
	r, err := httputils.RemoteQueryListOpts(req.Ctx(), req.Scope, "order_claim", v, req.QueryConfig.Fields, true)
	if err != nil {
		return err
	}
	var claim any
	if len(r.Data) > 0 {
		claim = r.Data[0]
	}
	var orderReturn any
	if c, ok := claim.(*dal.Record); ok {
		if rid, _ := c.Get("return_id"); rid != nil && rid != "" {
			if orderReturn, err = claims.QueryReturnDetails(req, rid); err != nil {
				return err
			}
		}
	}
	out := dal.NewRecord()
	out.Set("order_preview", preview)
	out.Set("claim", claim)
	if orderReturn != nil {
		out.Set("return", orderReturn)
	}
	return res.JSON(out)
}

// DELETE mirrors DELETE /admin/claims/:id/request.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if _, err := workflows.CancelBeginOrderClaimWorkflow.Run(req.Ctx(), req.Scope, any(req.Params["id"])); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", req.Params["id"])
	out.Set("object", "claim")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST, DELETE: DELETE}
