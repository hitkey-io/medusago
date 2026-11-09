// Package rules mirrors api/admin/tax-rates/[id]/rules/route.ts.
package rules

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/tax/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	taxrates "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/tax-rates"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	in := dal.NewRecord()
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		in.Set(k, v)
	}
	in.Set("tax_rate_id", req.Params["id"])
	in.Set("created_by", req.AuthString("actor_id"))
	if _, err := workflows.CreateTaxRateRulesWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{in}); err != nil {
		return err
	}
	v, err := taxrates.RefetchTaxRate(req, req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return taxrates.Respond(res, v)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
