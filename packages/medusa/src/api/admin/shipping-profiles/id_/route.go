// Package id_ mirrors api/admin/shipping-profiles/[id]/route.ts.
package id_

import (
	fulfillmentsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/workflows"
	profileworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/shipping-profile/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	shippingprofiles "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/shipping-profiles"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return shippingprofiles.Respond(req, res, req.Params["id"])
}

// POST mirrors POST (the raw body is the update, as upstream).
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	update, _ := req.Body.(*dal.Record)
	if update == nil {
		update = dal.NewRecord()
	}
	selector := dal.NewRecord()
	selector.Set("id", req.Params["id"])
	if _, err := workflows.UpdateShippingProfilesWorkflow.Run(req.Ctx(), req.Scope, fulfillmentsteps.SelectorUpdateInput{Selector: selector, Update: update}); err != nil {
		return err
	}
	return shippingprofiles.Respond(req, res, req.Params["id"])
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	svc := fulfillmentsteps.Fulfillment(req.Scope)
	if _, err := svc.Internal("ShippingProfile").Retrieve(req.Ctx(), id, utilsmodulessdk.FindConfig{}); err != nil {
		return err
	}
	if _, err := profileworkflows.DeleteShippingProfileWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "shipping_profile")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
