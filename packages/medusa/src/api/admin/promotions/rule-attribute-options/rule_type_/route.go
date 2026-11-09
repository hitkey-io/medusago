// Package rule_type_ mirrors api/admin/promotions/rule-attribute-options/[rule_type]/route.ts.
package rule_type_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	promoutils "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/promotions/utils"
)

func queryParam(req *medusahttp.MedusaRequest, k string) any {
	if req.Query == nil {
		return nil
	}
	v, _ := req.Query.Get(k)
	return v
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	ruleType := req.Params["rule_type"]
	if err := promoutils.ValidateRuleType(ruleType); err != nil {
		return err
	}
	attrs := promoutils.RuleAttributesMap(queryParam(req, "promotion_type"), queryParam(req, "application_method_type"), queryParam(req, "application_method_target_type"))[ruleType]
	out := dal.NewRecord()
	list := make([]any, len(attrs))
	for i, a := range attrs {
		list[i] = a
	}
	out.Set("attributes", list)
	return res.JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
