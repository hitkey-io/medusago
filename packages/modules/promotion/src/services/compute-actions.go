package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	computeactions "github.com/hitkey-io/medusago/packages/modules/promotion/src/utils/compute-actions"
)

// ComputeActionOptions mirrors ComputeActionOptions.
type ComputeActionOptions struct {
	PreventAutoPromotions bool
	SkipUsageLimitChecks  bool
}

func crecords(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		var out []*dal.Record
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

func cget(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func crec(r *dal.Record, k string) *dal.Record {
	v, _ := cget(r, k).(*dal.Record)
	return v
}

func cstr(r *dal.Record, k string) string {
	v, _ := cget(r, k).(string)
	return v
}

func ctruthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	}
	return true
}

// listActivePromotions mirrors listActivePromotions_.
func (s *PromotionModuleService) listActivePromotions(ctx context.Context, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	now := time.Now().UTC()
	a := dal.NewRecord()
	a.Set("campaign_id", nil)
	b := dal.NewRecord()
	for _, k := range filters.Keys() {
		v, _ := filters.Get(k)
		a.Set(k, v)
		b.Set(k, v)
	}
	or := func(field string, op string) *dal.Record {
		n := dal.NewRecord()
		n.Set(field, nil)
		c := dal.NewRecord()
		cmp := dal.NewRecord()
		cmp.Set(op, now)
		c.Set(field, cmp)
		r := dal.NewRecord()
		r.Set("$or", []any{n, c})
		return r
	}
	campaign := dal.NewRecord()
	if existing := crec(filters, "campaign"); existing != nil {
		for _, k := range existing.Keys() {
			v, _ := existing.Get(k)
			campaign.Set(k, v)
		}
	}
	campaign.Set("$and", []any{or("starts_at", "$lte"), or("ends_at", "$gt")})
	b.Set("campaign", campaign)
	f := dal.NewRecord()
	f.Set("status", "active")
	f.Set("$or", []any{a, b})
	return s.Internal("Promotion").List(ctx, f, config)
}

// pickValueFromObject mirrors pickValueFromObject (arrays are flattened).
func pickValueFromObject(path string, obj any) (any, bool) {
	segments := strings.Split(path, ".")
	result := obj
	for i, seg := range segments {
		r, ok := result.(*dal.Record)
		if !ok || r == nil {
			return nil, false
		}
		v, ok := r.Get(seg)
		if !ok || v == nil {
			return nil, false
		}
		result = v
		if i == len(segments)-1 {
			return result, true
		}
		if list, isList := asList(result); isList {
			sub := strings.Join(segments[i+1:], ".")
			out := []any{}
			for _, item := range list {
				val, ok := pickValueFromObject(sub, item)
				if l, isL := asList(val); isL {
					out = append(out, l...)
				} else if ok {
					out = append(out, val)
				} else {
					out = append(out, nil)
				}
			}
			return out, true
		}
		if _, isObj := result.(*dal.Record); !isObj {
			return result, true
		}
	}
	return result, true
}

func asList(v any) ([]any, bool) {
	switch t := v.(type) {
	case []any:
		return t, true
	case []*dal.Record:
		out := make([]any, len(t))
		for i, r := range t {
			out[i] = r
		}
		return out, true
	}
	return nil, false
}

func jsString(v any) string {
	switch t := v.(type) {
	case nil:
		return "undefined"
	case string:
		return t
	case float64:
		return totals.JSNumberToString(t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	}
	return fmt.Sprint(v)
}

// evaluateRuleValueCondition mirrors evaluateRuleValueCondition.
func evaluateRuleValueCondition(ruleValues []string, operator string, toCheck any, defined bool) bool {
	values, isList := asList(toCheck)
	if !isList {
		values = []any{toCheck}
		if !defined {
			values = []any{nil}
		}
	}
	if len(values) == 0 {
		return operator == "ne"
	}
	set := map[string]bool{}
	for _, v := range ruleValues {
		set[v] = true
	}
	switch operator {
	case "eq":
		for _, v := range values {
			if !set[jsString(v)] {
				return false
			}
		}
		return true
	case "in":
		for _, v := range values {
			if set[jsString(v)] {
				return true
			}
		}
		return false
	case "ne":
		for _, v := range values {
			if set[jsString(v)] {
				return false
			}
		}
		return true
	case "gt", "gte", "lt", "lte":
		cmp := map[string]func(a, b any) bool{"gt": totals.Gt, "gte": totals.Gte, "lt": totals.Lt, "lte": totals.Lte}[operator]
		for _, v := range values {
			ok := false
			for _, rv := range ruleValues {
				if cmp(v, rv) {
					ok = true
					break
				}
			}
			if !ok {
				return false
			}
		}
		return true
	}
	return false
}

// areRulesValidForContext mirrors areRulesValidForContext.
func areRulesValidForContext(rules []*dal.Record, ctx *dal.Record, scope string) bool {
	for _, rule := range rules {
		attr := cstr(rule, "attribute")
		values := crecords(cget(rule, "values"))
		if attr == "" || len(values) == 0 {
			return false
		}
		var valid []string
		for _, v := range values {
			if s, ok := cget(v, "value").(string); ok {
				valid = append(valid, s)
			}
		}
		if len(valid) == 0 {
			return false
		}
		switch scope {
		case "items":
			attr = strings.Replace(attr, "items.", "", 1)
		case "shipping_methods":
			attr = strings.Replace(attr, "shipping_methods.", "", 1)
		}
		v, ok := pickValueFromObject(attr, ctx)
		if !evaluateRuleValueCondition(valid, cstr(rule, "operator"), v, ok) {
			return false
		}
	}
	return true
}

func action(kv ...any) *dal.Record {
	r := dal.NewRecord()
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i].(string), kv[i+1])
	}
	return r
}

// computeActionForBudgetExceeded mirrors computeActionForBudgetExceeded.
func computeActionForBudgetExceeded(promotion *dal.Record, amount any, usage *dal.Record) *dal.Record {
	budget := crec(crec(promotion, "campaign"), "budget")
	if budget == nil {
		return nil
	}
	typ := cstr(budget, "type")
	if typ == "use_by_attribute" && usage == nil {
		return nil
	}
	spendByAttr := typ == "spend_by_attribute"
	var used any
	switch {
	case usage != nil:
		used = cget(usage, "used")
	case spendByAttr:
		used = 0
	default:
		used = cget(budget, "used")
		if used == nil {
			used = 0
		}
	}
	var total *totals.BN
	if typ == "spend" || spendByAttr {
		total = totals.Add(used, amount)
	} else {
		total = totals.Add(used, 1)
	}
	if limit := cget(budget, "limit"); ctruthy(limit) && totals.Gt(total, limit) {
		return action("action", "campaignBudgetExceeded", "code", cget(promotion, "code"))
	}
	return nil
}

const medusaEpsilon = "0.0001"

func lineItemSubtotal(item *dal.Record) *totals.BN {
	return totals.Div(cget(item, "subtotal"), cget(item, "quantity"))
}

func appliedValueInBase(p promoCalc, item *dal.Record) any {
	if !p.isTaxInclusive {
		return p.appliedValue
	}
	return totals.Mult(p.appliedValue, totals.Div(cget(item, "original_total"), cget(item, "subtotal")))
}

type promoCalc struct {
	value, appliedValue any
	isTaxInclusive      bool
	maxQuantity         any
	typ, allocation     string
}

func getPromotionValue(p promoCalc, lineItemAmount, lineItemsAmount any, item *dal.Record) *totals.BN {
	if p.typ == "percentage" {
		return totals.Mult(totals.Div(p.value, 100), lineItemAmount)
	}
	if p.allocation == "across" {
		forItem := totals.Mult(totals.Div(lineItemAmount, lineItemsAmount), p.value)
		if totals.Lte(forItem, lineItemAmount) {
			return forItem
		}
		pct := totals.Div(totals.Mult(lineItemAmount, 100), forItem)
		return totals.Mult(forItem, totals.Div(pct, 100))
	}
	maxQ := p.maxQuantity
	if maxQ == nil {
		maxQ = 1
	}
	maxAmount := totals.Mult(p.value, totals.Min(cget(item, "quantity"), maxQ))
	return totals.Min(maxAmount, lineItemAmount)
}

// calculateAdjustmentAmountFromPromotion mirrors calculateAdjustmentAmountFromPromotion.
func calculateAdjustmentAmountFromPromotion(item *dal.Record, p promoCalc, lineItemsAmount any) *totals.BN {
	base := cget(item, "subtotal")
	if p.isTaxInclusive {
		base = cget(item, "original_total")
	}
	if p.allocation == "across" {
		quantity := cget(item, "quantity")
		if ctruthy(p.maxQuantity) && ctruthy(quantity) {
			quantity = totals.Min(quantity, p.maxQuantity)
		}
		per := lineItemSubtotal(item)
		if p.isTaxInclusive {
			per = totals.Div(cget(item, "original_total"), cget(item, "quantity"))
		}
		applicable := totals.Sub(totals.Mult(per, quantity), appliedValueInBase(p, item))
		if totals.Lte(applicable, medusaEpsilon) {
			return totals.Convert(0)
		}
		ret := totals.Min(getPromotionValue(p, applicable, lineItemsAmount, item), applicable)
		if totals.Lte(ret, medusaEpsilon) {
			return totals.Convert(0)
		}
		return ret
	}
	remaining := totals.Sub(base, appliedValueInBase(p, item))
	itemAmount := totals.Div(base, cget(item, "quantity"))
	maxQ := p.maxQuantity
	if maxQ == nil {
		maxQ = 1
	}
	applicable := totals.Min(remaining, totals.Mult(itemAmount, maxQ))
	if totals.Lte(applicable, medusaEpsilon) {
		return totals.Convert(0)
	}
	ret := totals.Min(getPromotionValue(p, applicable, lineItemsAmount, item), applicable)
	if totals.Lte(ret, medusaEpsilon) {
		return totals.Convert(0)
	}
	return ret
}

func getValidItemsForPromotion(items []*dal.Record, promotion *dal.Record) []*dal.Record {
	am := crec(promotion, "application_method")
	if len(items) == 0 || am == nil {
		return nil
	}
	isShipping := cstr(am, "target_type") == "shipping_methods"
	targetRules := crecords(cget(am, "target_rules"))
	var out []*dal.Record
	for _, item := range items {
		if item == nil {
			continue
		}
		if isShipping && len(targetRules) == 0 {
			if item.Has("subtotal") && totals.Gt(cget(item, "subtotal"), 0) {
				out = append(out, item)
			}
			continue
		}
		if item.Has("is_discountable") && !ctruthy(cget(item, "is_discountable")) {
			continue
		}
		if !item.Has("subtotal") || totals.Lte(cget(item, "subtotal"), 0) {
			continue
		}
		if !isShipping && !item.Has("quantity") {
			continue
		}
		if len(targetRules) == 0 || areRulesValidForContext(targetRules, item, "items") {
			out = append(out, item)
		}
	}
	return out
}

func number(v any) float64 {
	f, _ := totals.Convert(v).Float64()
	return f
}

// applyPromotionToItems mirrors applyPromotionToItems.
func applyPromotionToItems(promotion *dal.Record, items []*dal.Record, applied map[string]any, allocationOverride string, usage *dal.Record) []*dal.Record {
	am := crec(promotion, "application_method")
	if am == nil {
		return nil
	}
	allocation := cstr(am, "allocation")
	if allocation == "" {
		allocation = allocationOverride
	}
	target := cstr(am, "target_type")
	if len(items) == 0 || target == "" {
		return nil
	}
	var actions []*dal.Record
	applicable := getValidItemsForPromotion(items, promotion)
	if len(applicable) == 0 {
		return actions
	}
	if allocation == "once" {
		sort.SliceStable(applicable, func(i, j int) bool {
			return totals.Lt(lineItemSubtotal(applicable[i]), lineItemSubtotal(applicable[j]))
		})
	}
	inclusive := ctruthy(cget(promotion, "is_tax_inclusive"))
	value := cget(am, "value")
	if value == nil {
		value = 0
	}
	maxQuantity := cget(am, "max_quantity")
	remaining := number(maxQuantity)
	appliedOf := func(id string) any {
		if v, ok := applied[id]; ok {
			return v
		}
		return 0
	}
	lineItemsAmount := totals.Convert(0)
	if allocation == "across" {
		for _, item := range applicable {
			base := cget(item, "subtotal")
			if inclusive {
				base = cget(item, "original_total")
			}
			appliedExcl := appliedOf(cstr(item, "id"))
			var inBase any = appliedExcl
			if inclusive {
				inBase = totals.Mult(appliedExcl, totals.Div(cget(item, "original_total"), cget(item, "subtotal")))
			}
			lineItemsAmount = totals.Sub(totals.Add(lineItemsAmount, base), inBase)
		}
		if totals.Lte(lineItemsAmount, 0) {
			return actions
		}
	}
	for _, item := range applicable {
		if allocation == "once" && remaining <= 0 {
			break
		}
		base := cget(item, "subtotal")
		if inclusive {
			base = cget(item, "original_total")
		}
		if totals.Lte(base, 0) {
			continue
		}
		id := cstr(item, "id")
		appliedValue := appliedOf(id)
		var effectiveMax any = maxQuantity
		effectiveAllocation := allocation
		if allocation == "once" {
			effectiveMax = minFloat(remaining, number(cget(item, "quantity")))
			effectiveAllocation = "each"
		}
		amount := calculateAdjustmentAmountFromPromotion(item, promoCalc{value: value, appliedValue: appliedValue, isTaxInclusive: inclusive, maxQuantity: effectiveMax, typ: cstr(am, "type"), allocation: effectiveAllocation}, lineItemsAmount)
		if totals.Lte(amount, 0) {
			continue
		}
		if a := computeActionForBudgetExceeded(promotion, amount, usage); a != nil {
			actions = append(actions, a)
			continue
		}
		var appliedExcl any = amount
		if inclusive {
			appliedExcl = totals.Mult(amount, totals.Div(cget(item, "subtotal"), cget(item, "original_total")))
		}
		applied[id] = totals.Add(appliedValue, appliedExcl)
		if allocation == "once" {
			remaining -= minFloat(number(effectiveMax), number(cget(item, "quantity")))
		}
		if target == "items" || target == "order" {
			actions = append(actions, action("action", "addItemAdjustment", "item_id", cget(item, "id"), "amount", amount, "code", cget(promotion, "code"), "is_tax_inclusive", cget(promotion, "is_tax_inclusive")))
		}
	}
	return actions
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// applyPromotionToShippingMethods mirrors applyPromotionToShippingMethods.
func applyPromotionToShippingMethods(promotion *dal.Record, methods []*dal.Record, applied map[string]any, usage *dal.Record) []*dal.Record {
	am := crec(promotion, "application_method")
	allocation := cstr(am, "allocation")
	var actions []*dal.Record
	remaining := number(cget(am, "max_quantity"))
	appliedOf := func(id string) any {
		if v, ok := applied[id]; ok {
			return v
		}
		return 0
	}
	value := cget(am, "value")
	if value == nil {
		value = 0
	}
	if allocation == "each" || allocation == "once" {
		for _, m := range methods {
			if allocation == "once" && remaining <= 0 {
				break
			}
			if !ctruthy(cget(m, "subtotal")) {
				continue
			}
			id := cstr(m, "id")
			appliedValue := appliedOf(id)
			promotionValue := totals.Convert(value)
			applicable := totals.Sub(cget(m, "subtotal"), appliedValue)
			if cstr(am, "type") == "percentage" {
				promotionValue = totals.Mult(totals.Div(promotionValue, 100), applicable)
			}
			amount := totals.Min(promotionValue, applicable)
			if totals.Lte(amount, 0) {
				continue
			}
			if a := computeActionForBudgetExceeded(promotion, amount, usage); a != nil {
				actions = append(actions, a)
				continue
			}
			applied[id] = totals.Add(appliedValue, amount)
			if allocation == "once" {
				remaining--
			}
			actions = append(actions, action("action", "addShippingMethodAdjustment", "shipping_method_id", cget(m, "id"), "amount", amount, "code", cget(promotion, "code")))
		}
	}
	if allocation == "across" {
		total := totals.Convert(0)
		for _, m := range methods {
			sub := cget(m, "subtotal")
			if sub == nil {
				sub = 0
			}
			total = totals.Add(total, totals.Sub(sub, appliedOf(cstr(m, "id"))))
		}
		if totals.Lte(total, 0) {
			return actions
		}
		for _, m := range methods {
			if !ctruthy(cget(m, "subtotal")) {
				continue
			}
			id := cstr(m, "id")
			appliedValue := appliedOf(id)
			applicable := totals.Sub(cget(m, "subtotal"), appliedValue)
			promotionValue := totals.Mult(totals.Div(applicable, total), value)
			if cstr(am, "type") == "percentage" {
				promotionValue = totals.Mult(totals.Div(value, 100), applicable)
			}
			amount := totals.Min(promotionValue, applicable)
			if totals.Lte(amount, 0) {
				continue
			}
			if a := computeActionForBudgetExceeded(promotion, amount, usage); a != nil {
				actions = append(actions, a)
				continue
			}
			applied[id] = totals.Add(appliedValue, amount)
			actions = append(actions, action("action", "addShippingMethodAdjustment", "shipping_method_id", cget(m, "id"), "amount", amount, "code", cget(promotion, "code")))
		}
	}
	return actions
}

// ComputeActions mirrors computeActions.
func (s *PromotionModuleService) ComputeActions(ctx context.Context, promotionCodes []string, applicationContext *dal.Record, options ComputeActionOptions) ([]*dal.Record, error) {
	actions := []*dal.Record{}
	items := crecords(cget(applicationContext, "items"))
	methods := crecords(cget(applicationContext, "shipping_methods"))
	type adj struct{ items, shipping []*dal.Record }
	var order []string
	byCode := map[string]*adj{}
	collect := func(lines []*dal.Record, shipping bool) {
		for _, line := range lines {
			for _, a := range crecords(cget(line, "adjustments")) {
				code, ok := cget(a, "code").(string)
				if !ok {
					continue
				}
				if byCode[code] == nil {
					byCode[code] = &adj{}
					order = append(order, code)
				}
				if shipping {
					byCode[code].shipping = append(byCode[code].shipping, a)
				} else {
					byCode[code].items = append(byCode[code].items, a)
				}
			}
		}
	}
	collect(items, false)
	collect(methods, true)

	codes := []any{}
	seen := map[string]bool{}
	for _, c := range append(append([]string{}, promotionCodes...), order...) {
		if !seen[c] {
			seen[c] = true
			codes = append(codes, c)
		}
	}
	codeFilter := dal.NewRecord()
	codeFilter.Set("code", codes)
	filter := codeFilter
	if !options.PreventAutoPromotions {
		// the automatic promotions whose rules the context can satisfy
		// (prefiltered in SQL)
		rulePrefilter, ok, err := computeactions.BuildPromotionRuleQueryFilterFromContext(ctx, s.DB(), applicationContext)
		if err != nil {
			return nil, err
		}
		auto := dal.NewRecord()
		if ok {
			automatic := dal.NewRecord()
			automatic.Set("is_automatic", true)
			raw := dal.NewRecord()
			raw.Set("__rule_prefilter", rulePrefilter)
			pf := dal.NewRecord()
			pf.Set("$and", []any{automatic, raw})
			rows, err := s.Internal("Promotion").List(ctx, pf, utilsmodulessdk.FindConfig{Select: []string{"id"}})
			if err != nil {
				return nil, err
			}
			ids := []any{}
			for _, r := range rows {
				ids = append(ids, cget(r, "id"))
			}
			in := dal.NewRecord()
			in.Set("$in", ids)
			auto.Set("id", in)
		} else {
			auto.Set("is_automatic", true)
		}
		filter = dal.NewRecord()
		filter.Set("$or", []any{codeFilter, auto})
	}
	// ordered by the application method value in SQL: promotions with equal
	// values keep the order of the query plan, like upstream
	promotions, err := s.listActivePromotions(ctx, filter, utilsmodulessdk.FindConfig{
		Order: []dal.OrderBy{{Field: "application_method.value", Direction: "DESC"}},
		Relations: []string{
			"application_method", "application_method.target_rules", "application_method.target_rules.values",
			"application_method.buy_rules", "application_method.buy_rules.values", "rules", "rules.values", "campaign", "campaign.budget",
		}})
	if err != nil {
		return nil, err
	}
	byPromoCode := map[string]*dal.Record{}
	automatic := map[string]bool{}
	for _, p := range promotions {
		byPromoCode[cstr(p, "code")] = p
		if ctruthy(cget(p, "is_automatic")) {
			automatic[cstr(p, "code")] = true
		}
	}
	for _, code := range order {
		// undefined line ids are dropped (JSON)
		for _, a := range byCode[code].items {
			r := action("action", "removeItemAdjustment", "adjustment_id", cget(a, "id"), "code", code)
			if a.Has("item_id") {
				r.Set("item_id", cget(a, "item_id"))
			}
			actions = append(actions, r)
		}
		for _, a := range byCode[code].shipping {
			r := action("action", "removeShippingMethodAdjustment", "adjustment_id", cget(a, "id"), "code", code)
			if a.Has("shipping_method_id") {
				r.Set("shipping_method_id", cget(a, "shipping_method_id"))
			}
			actions = append(actions, r)
		}
	}
	requested := map[string]bool{}
	for _, c := range promotionCodes {
		requested[c] = true
	}
	var toApply []*dal.Record
	for _, p := range promotions {
		if requested[cstr(p, "code")] || automatic[cstr(p, "code")] {
			toApply = append(toApply, p)
		}
	}
	sort.SliceStable(toApply, func(i, j int) bool {
		return sortByBuyGetType(toApply[i], toApply[j]) < 0
	})
	eligibleBuyItemMap := newEligibleItemsByCode()
	eligibleTargetItemMap := newEligibleItemsByCode()
	applied := map[string]any{}
	for _, pa := range toApply {
		promotion := byPromoCode[cstr(pa, "code")]
		am := crec(promotion, "application_method")
		if am == nil {
			continue
		}
		budget := crec(crec(promotion, "campaign"), "budget")
		budgetType := cstr(budget, "type")
		var usage *dal.Record
		if !options.SkipUsageLimitChecks && (budgetType == "use_by_attribute" || budgetType == "spend_by_attribute") {
			attribute := cstr(budget, "attribute")
			usageCtx := map[string]any{
				"customer_id":    firstDefined(cget(applicationContext, "customer_id"), cget(crec(applicationContext, "customer"), "id")),
				"customer_email": cget(applicationContext, "email"),
			}
			value := usageCtx[attribute]
			if !ctruthy(value) {
				if ctruthy(cget(promotion, "is_automatic")) {
					continue
				}
				return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Attribute value for \"%s\" is required by promotion campaing budget", attribute))
			}
			f := dal.NewRecord()
			f.Set("budget_id", cget(budget, "id"))
			f.Set("attribute_value", value)
			rows, err := s.Internal("CampaignBudgetUsage").List(ctx, f, utilsmodulessdk.FindConfig{})
			if err != nil {
				return nil, err
			}
			if len(rows) > 0 {
				usage = rows[0]
			}
			if budgetType == "use_by_attribute" && usage != nil {
				if a := computeActionForBudgetExceeded(promotion, 1, usage); a != nil {
					actions = append(actions, a)
					continue
				}
			}
		}
		if !options.SkipUsageLimitChecks {
			if limit, ok := cget(promotion, "limit").(float64); ok {
				used := number(cget(promotion, "used"))
				if used >= limit {
					actions = append(actions, action("action", "promotionLimitExceeded", "code", cget(promotion, "code")))
					continue
				}
			}
		}
		currencyValid := !ctruthy(cget(am, "currency_code")) || cget(applicationContext, "currency_code") == cget(am, "currency_code")
		if !areRulesValidForContext(crecords(cget(promotion, "rules")), applicationContext, "order") || !currencyValid {
			continue
		}
		switch cstr(promotion, "type") {
		case "buyget":
			_, itemsDefined := applicationContext.Get("items")
			itemsDefined = itemsDefined && cget(applicationContext, "items") != nil
			buyGetActions, err := getComputedActionsForBuyGet(promotion, items, itemsDefined, applied, eligibleBuyItemMap, eligibleTargetItemMap, usage)
			if err != nil {
				return nil, err
			}
			actions = append(actions, buyGetActions...)
		case "standard":
			target := cstr(am, "target_type")
			override := ""
			if target == "order" {
				override = "across"
			}
			if target == "order" || target == "items" {
				if _, ok := applicationContext.Get("items"); !ok || cget(applicationContext, "items") == nil {
					return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, `"items" should be present as an array in the context for computeActions`)
				}
				actions = append(actions, applyPromotionToItems(promotion, items, applied, override, usage)...)
			}
			if target == "shipping_methods" {
				if _, ok := applicationContext.Get("shipping_methods"); !ok || cget(applicationContext, "shipping_methods") == nil {
					return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, `"shipping_methods" should be present as an array in the context for computeActions`)
				}
				var applicableMethods []*dal.Record
				for _, m := range methods {
					if areRulesValidForContext(crecords(cget(am, "target_rules")), m, "shipping_methods") {
						applicableMethods = append(applicableMethods, m)
					}
				}
				if cstr(am, "allocation") == "once" {
					sort.SliceStable(applicableMethods, func(i, j int) bool {
						return totals.Lt(firstDefined(cget(applicableMethods[i], "subtotal"), 0), firstDefined(cget(applicableMethods[j], "subtotal"), 0))
					})
				}
				actions = append(actions, applyPromotionToShippingMethods(promotion, applicableMethods, applied, usage)...)
			}
		}
	}
	for _, a := range actions {
		if v, ok := a.Get("amount"); ok && v != nil {
			a.Set("amount", totals.NewBN(v))
		}
	}
	return actions, nil
}

func firstDefined(vs ...any) any {
	for _, v := range vs {
		if v != nil {
			return v
		}
	}
	return nil
}
