// Package id_ mirrors api/admin/shipping-option-types/[id]/route.ts.
package id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/shipping-options/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	shippingoptiontypes "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/shipping-option-types"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return shippingoptiontypes.Respond(req, res, req.Params["id"])
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	existing, err := shippingoptiontypes.Refetch(req, id, []string{"id"})
	if err != nil {
		return err
	}
	if existing == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`Shipping option type with id "%s" not found`, id))
	}
	body, _ := req.ValidatedBody.(*dal.Record)
	selector := dal.NewRecord()
	selector.Set("id", id)
	result, err := workflows.UpdateShippingOptionTypesWorkflow.Run(req.Ctx(), req.Scope, workflows.UpdateShippingOptionTypesInput{Selector: selector, Update: body})
	if err != nil {
		return err
	}
	var rid any
	if len(result) > 0 {
		rid, _ = result[0].Get("id")
	}
	return shippingoptiontypes.Respond(req, res, rid)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeleteShippingOptionTypesWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "shipping_option_type")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
