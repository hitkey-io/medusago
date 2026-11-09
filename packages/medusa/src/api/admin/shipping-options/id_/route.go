// Package id_ mirrors api/admin/shipping-options/[id]/route.ts.
package id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	shippingoptions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/shipping-options"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	v, err := shippingoptions.RefetchShippingOption(req, req.Params["id"])
	if err != nil {
		return err
	}
	if v == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Shipping Option with id: %s not found", req.Params["id"]))
	}
	out := dal.NewRecord()
	out.Set("shipping_option", v)
	return res.JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	in := dal.NewRecord()
	in.Set("id", req.Params["id"])
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		in.Set(k, v)
	}
	result, err := workflows.UpdateShippingOptionsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{in})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	return shippingoptions.Respond(req, res, id)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeleteShippingOptionsWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "shipping_option")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
