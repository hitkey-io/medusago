// Package id_ mirrors api/admin/price-preferences/[id]/route.ts.
package id_

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/pricing/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/pricing/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	pricepreferences "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-preferences"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return pricepreferences.Respond(req, res, req.Params["id"])
}

// POST mirrors POST (the raw request body is the update, as upstream).
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	update, _ := req.Body.(*dal.Record)
	if update == nil {
		update = dal.NewRecord()
	}
	selector := dal.NewRecord()
	selector.Set("id", []any{id})
	if _, err := workflows.UpdatePricePreferencesWorkflow.Run(req.Ctx(), req.Scope, steps.UpdatePricePreferencesInput{Selector: selector, Update: update}); err != nil {
		return err
	}
	return pricepreferences.Respond(req, res, id)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeletePricePreferencesWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "price_preference")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
