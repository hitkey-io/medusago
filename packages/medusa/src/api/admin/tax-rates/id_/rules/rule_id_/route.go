// Package rule_id_ mirrors api/admin/tax-rates/[id]/rules/[rule_id]/route.ts.
package rule_id_

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/tax/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	taxrates "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/tax-rates"
)

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	ruleID := req.Params["rule_id"]
	if _, err := workflows.DeleteTaxRateRulesWorkflow.Run(req.Ctx(), req.Scope, []any{ruleID}); err != nil {
		return err
	}
	v, err := taxrates.RefetchTaxRate(req, req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", ruleID)
	out.Set("object", "tax_rate_rule")
	out.Set("deleted", true)
	if v != nil {
		out.Set("parent", v)
	}
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{DELETE: DELETE}
