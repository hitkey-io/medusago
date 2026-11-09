// Package rule_attribute_id_ mirrors
// api/admin/promotions/rule-value-options/[rule_type]/[rule_attribute_id]/route.ts
// (without the index engine, which is behind a feature flag).
package rule_attribute_id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	promoutils "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/promotions/utils"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	ruleType, attributeID := req.Params["rule_type"], req.Params["rule_attribute_id"]
	cfg, hasCfg := promoutils.RuleQueryConfigurations[attributeID]
	filters := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			filters.Set(k, v)
		}
	}
	if err := promoutils.ValidateRuleType(ruleType); err != nil {
		return err
	}
	targetType, _ := filters.Get("application_method_target_type")
	if err := promoutils.ValidateRuleAttribute(ruleType, attributeID, nil, nil, targetType); err != nil {
		return err
	}
	if !hasCfg {
		return errTypeError("Cannot read properties of undefined (reading 'valueAttr')")
	}
	if v, ok := filters.Get("value"); ok && v != nil && v != "" {
		filters.Set(cfg.ValueAttr, v)
		filters.Delete("value")
	}
	if v, ok := filters.Get("application_method_target_type"); ok && v != nil && v != "" {
		filters.Delete("application_method_target_type")
	}
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: cfg.EntryPoint, Fields: []string{cfg.LabelAttr, cfg.ValueAttr}, Filters: filters, Pagination: req.QueryConfig.Pagination}, joiner.Options{})
	if err != nil {
		return err
	}
	values := make([]any, 0, len(r.Data))
	for _, row := range r.Data {
		rr, _ := row.(*dal.Record)
		v := dal.NewRecord()
		label, _ := rr.Get(cfg.LabelAttr)
		value, _ := rr.Get(cfg.ValueAttr)
		v.Set("label", label)
		v.Set("value", value)
		values = append(values, v)
	}
	out := dal.NewRecord()
	out.Set("values", values)
	if r.Metadata != nil {
		for _, pair := range [][2]string{{"count", "count"}, {"offset", "skip"}, {"limit", "take"}} {
			if v, ok := r.Metadata.Get(pair[1]); ok {
				out.Set(pair[0], v)
			}
		}
	}
	return res.JSON(out)
}

type typeError string

func (e typeError) Error() string { return string(e) }

func errTypeError(msg string) error { return typeError(msg) }

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
