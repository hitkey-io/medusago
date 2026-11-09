// Package id_ mirrors api/admin/tax-rates/[id]/route.ts.
package id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/tax/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	taxrates "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/tax-rates"
)

// GET mirrors GET: the id is passed as a top-level variable (not a filter),
// as upstream does.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	vars := dal.NewRecord()
	vars.Set("id", req.Params["id"])
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "tax_rate", vars, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	var v any
	if len(r.Data) > 0 {
		v = r.Data[0]
	}
	return taxrates.Respond(res, v)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	existing, err := taxrates.RefetchTaxRate(req, id, []string{"id"})
	if err != nil {
		return err
	}
	if existing == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`Tax rate with id "%s" not found`, id))
	}
	body, _ := req.ValidatedBody.(*dal.Record)
	update := dal.NewRecord()
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		update.Set(k, v)
	}
	update.Set("updated_by", req.AuthString("actor_id"))
	selector := dal.NewRecord()
	selector.Set("id", id)
	if _, err := workflows.UpdateTaxRatesWorkflow.Run(req.Ctx(), req.Scope, workflows.UpdateTaxRatesInput{Selector: selector, Update: update}); err != nil {
		return err
	}
	v, err := taxrates.RefetchTaxRate(req, id, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return taxrates.Respond(res, v)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeleteTaxRatesWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "tax_rate")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
