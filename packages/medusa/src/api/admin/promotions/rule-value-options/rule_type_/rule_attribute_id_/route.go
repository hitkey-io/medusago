// Package rule_attribute_id_ mirrors
// api/admin/promotions/rule-value-options/[rule_type]/[rule_attribute_id]/route.ts
// (query.index for products with the index_engine flag).
package rule_attribute_id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	promoutils "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/promotions/utils"
	featureflags "github.com/hitkey-io/medusago/packages/medusa/src/feature-flags"
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
	useIndexEngine := featureflags.IsFeatureEnabled(req.Scope, featureflags.IndexEngine.Key) && cfg.EntryPoint == "product"
	if useIndexEngine {
		indexFilters := dal.NewRecord()
		for _, k := range filters.Keys() {
			v, _ := filters.Get(k)
			indexFilters.Set(k, v)
		}
		if q, ok := indexFilters.Get("q"); ok && q != nil && q != "" && q != false {
			if v, _ := indexFilters.Get("variants"); v == nil {
				indexFilters.Set("variants", dal.NewRecord())
			}
		}
		r, err := httputils.QueryOf(req.Scope).Index(req.Ctx(), query.GraphInput{Entity: cfg.EntryPoint, Fields: []string{cfg.LabelAttr, cfg.ValueAttr}, Filters: indexFilters, Pagination: req.QueryConfig.Pagination}, nil, joiner.Options{})
		if err != nil {
			return err
		}
		out := dal.NewRecord()
		out.Set("values", labelValues(r.Data, cfg.LabelAttr, cfg.ValueAttr))
		var estimate, skip, take any
		if r.Metadata != nil {
			estimate, _ = r.Metadata.Get("estimate_count")
			skip, _ = r.Metadata.Get("skip")
			take, _ = r.Metadata.Get("take")
		}
		out.Set("count", estimate)
		out.Set("estimate_count", estimate)
		out.Set("offset", skip)
		out.Set("limit", take)
		return res.JSON(out)
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

func labelValues(rows []any, labelAttr, valueAttr string) []any {
	values := make([]any, 0, len(rows))
	for _, row := range rows {
		rr, _ := row.(*dal.Record)
		v := dal.NewRecord()
		var label, value any
		if rr != nil {
			label, _ = rr.Get(labelAttr)
			value, _ = rr.Get(valueAttr)
		}
		v.Set("label", label)
		v.Set("value", value)
		values = append(values, v)
	}
	return values
}
