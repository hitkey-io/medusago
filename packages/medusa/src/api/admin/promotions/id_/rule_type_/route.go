// Package rule_type_ mirrors api/admin/promotions/[id]/[rule_type]/route.ts.
package rule_type_

import (
	"errors"
	"strings"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	promoutils "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/promotions/utils"
)

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func or(a, b any) any {
	switch t := a.(type) {
	case nil:
		return b
	case string:
		if t == "" {
			return b
		}
	}
	return a
}

func list(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case []*dal.Record:
		out := make([]any, len(t))
		for i, r := range t {
			out[i] = r
		}
		return out
	}
	return nil
}

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case bool:
		return t
	case float64:
		return t != 0
	}
	return true
}

func queryParam(req *medusahttp.MedusaRequest, k string) any {
	if req.Query == nil {
		return nil
	}
	v, _ := req.Query.Get(k)
	return v
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id, ruleType := req.Params["id"], req.Params["rule_type"]
	if err := promoutils.ValidateRuleType(ruleType); err != nil {
		return err
	}
	dasherized := strings.Join(strings.Split(ruleType, "-"), "_")
	vars := dal.NewRecord()
	vars.Set("id", id)
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "promotion", vars, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	var promotion *dal.Record
	if len(r.Data) > 0 {
		promotion, _ = r.Data[0].(*dal.Record)
	}
	am, _ := get(promotion, "application_method").(*dal.Record)
	attrs, ok := promoutils.RuleAttributesMap(
		or(get(promotion, "type"), queryParam(req, "promotion_type")),
		or(get(am, "type"), queryParam(req, "application_method_type")),
		or(get(am, "target_type"), queryParam(req, "application_method_target_type")),
	)[ruleType]
	if !ok {
		return errors.New("Cannot read properties of undefined (reading 'filter')")
	}
	var promotionRules []any
	switch dasherized {
	case "rules":
		promotionRules = append(promotionRules, list(get(promotion, "rules"))...)
	case "target_rules":
		promotionRules = append(promotionRules, list(get(am, "target_rules"))...)
	case "buy_rules":
		promotionRules = append(promotionRules, list(get(am, "buy_rules"))...)
	}
	transformed := []any{}
	for _, d := range attrs {
		if !truthy(get(d, "disguised")) {
			continue
		}
		attrID, _ := get(d, "id").(string)
		value := get(am, attrID)
		var defined bool
		if am != nil {
			_, defined = am.Get(attrID)
		}
		var values any
		switch {
		case get(d, "field_type") == "number":
			values = value
		case truthy(value):
			v := dal.NewRecord()
			v.Set("label", value)
			v.Set("value", value)
			values = []any{v}
		default:
			values = []any{}
		}
		required := true
		if v, ok := get(d, "required").(bool); ok {
			required = v
		}
		if !required && !truthy(value) {
			continue
		}
		t := dal.NewRecord()
		for _, k := range d.Keys() {
			if k == "id" || k == "value" {
				continue
			}
			v, _ := d.Get(k)
			t.Set(k, v)
		}
		t.Set("attribute", attrID)
		t.Set("attribute_label", get(d, "label"))
		t.Set("operator", "eq")
		t.Set("operator_label", get(promoutils.OperatorsMap["eq"], "label"))
		if values != nil || defined || get(d, "field_type") != "number" {
			t.Set("values", values)
		}
		transformed = append(transformed, t)
	}
	for _, x := range append(append([]any{}, promotionRules...), transformed...) {
		rule, _ := x.(*dal.Record)
		if rule == nil {
			continue
		}
		var current *dal.Record
		for _, a := range attrs {
			if get(a, "value") == get(rule, "attribute") {
				current = a
				break
			}
		}
		if current == nil {
			continue
		}
		cfg, ok := promoutils.RuleQueryConfigurations[get(current, "id").(string)]
		if !ok {
			continue
		}
		var values []any
		for _, v := range list(get(rule, "values")) {
			values = append(values, get(v.(*dal.Record), "value"))
		}
		f := dal.NewRecord()
		f.Set(cfg.ValueAttr, values)
		qv := dal.NewRecord()
		qv.Set("filters", f)
		rows, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, cfg.EntryPoint, qv, []string{cfg.LabelAttr, cfg.ValueAttr})
		if err != nil {
			return err
		}
		labels := map[any]any{}
		for _, row := range rows.Data {
			rr, _ := row.(*dal.Record)
			labels[get(rr, cfg.ValueAttr)] = get(rr, cfg.LabelAttr)
		}
		if vs := list(get(rule, "values")); vs != nil {
			mapped := make([]any, len(vs))
			for i, v := range vs {
				vr, _ := v.(*dal.Record)
				n := dal.NewRecord()
				n.Set("value", get(vr, "value"))
				label := labels[get(vr, "value")]
				if !truthy(label) {
					label = get(vr, "value")
				}
				n.Set("label", label)
				mapped[i] = n
			}
			rule.Set("values", mapped)
		}
		if !truthy(get(current, "hydrate")) {
			t := dal.NewRecord()
			for _, k := range current.Keys() {
				v, _ := current.Get(k)
				t.Set(k, v)
			}
			for _, k := range rule.Keys() {
				v, _ := rule.Get(k)
				t.Set(k, v)
			}
			t.Set("attribute_label", get(current, "label"))
			op := get(rule, "operator")
			if label := get(promoutils.OperatorsMap[str(op)], "label"); label != nil {
				t.Set("operator_label", label)
			} else {
				t.Set("operator_label", op)
			}
			transformed = append(transformed, t)
		}
	}
	out := dal.NewRecord()
	out.Set("rules", transformed)
	return res.JSON(out)
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
