// Package id_ mirrors api/admin/tax-regions/[id]/route.ts.
package id_

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/tax/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	taxregions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/tax-regions"
)

// GET mirrors GET (undefined when missing).
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return taxregions.Respond(req, res, req.Params["id"])
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	body, _ := req.ValidatedBody.(*dal.Record)
	in := dal.NewRecord()
	in.Set("id", id)
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		in.Set(k, v)
	}
	if _, err := workflows.UpdateTaxRegionsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{in}); err != nil {
		return err
	}
	return taxregions.RespondGraph(req, res, id)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeleteTaxRegionsWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "tax_region")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
