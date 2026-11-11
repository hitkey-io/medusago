package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

var (
	allTargetTypes               = []string{"order", "shipping_methods", "items"}
	allMethodTypes               = []string{"fixed", "percentage"}
	allAllocationTypes           = []string{"each", "across", "once"}
	allowedAllocationTargetTypes = []string{"shipping_methods", "items"}
	allowedAllocationTypes       = []string{"across", "each", "once"}
	allowedAllocationForQuantity = []string{"each", "once"}
	ruleOperators                = []string{"gte", "lte", "gt", "lt", "eq", "ne", "in"}
)

func contains(list []string, v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// or mirrors `a || b`.
func or(a, b any) any {
	if jsTruthy(a) {
		return a
	}
	return b
}

func badData(format string, args ...any) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(format, args...))
}

// validateApplicationMethodAttributes mirrors validateApplicationMethodAttributes.
func validateApplicationMethodAttributes(data *dal.Record, promotion *dal.Record) error {
	am, _ := get(promotion, "application_method").(*dal.Record)
	buyRulesMinQuantity := or(get(data, "buy_rules_min_quantity"), get(am, "buy_rules_min_quantity"))
	applyToQuantity := or(get(data, "apply_to_quantity"), get(am, "apply_to_quantity"))
	targetType := or(get(data, "target_type"), get(am, "target_type"))
	methodType := or(get(data, "type"), get(am, "type"))
	var value any = float64(0)
	if v := get(data, "value"); v != nil {
		value = v
	} else if v := get(am, "value"); v != nil {
		value = v
	}
	maxQuantity := or(get(data, "max_quantity"), get(am, "max_quantity"))
	allocation := or(get(data, "allocation"), get(am, "allocation"))

	if methodType == "percentage" && (totals.Lte(value, 0) || totals.Gt(value, 100)) {
		return badData("Application Method value should be a percentage number between 0 and 100")
	}
	if get(promotion, "type") == "buyget" {
		if !isPresent(applyToQuantity) {
			return badData("apply_to_quantity is a required field for Promotion type of buyget")
		}
		if !isPresent(buyRulesMinQuantity) {
			return badData("buy_rules_min_quantity is a required field for Promotion type of buyget")
		}
		if !isPresent(maxQuantity) {
			return badData("application_method.max_quantity is a required field for Promotion type of buyget")
		}
		if totals.Lt(maxQuantity, applyToQuantity) {
			return badData("max_quantity (%s) must be greater than or equal to apply_to_quantity (%s) for BUYGET promotions.", jsNum(maxQuantity), jsNum(applyToQuantity))
		}
	}
	if allocation == "across" && isPresent(maxQuantity) {
		return badData("application_method.max_quantity is not allowed to be set for allocation (across)")
	}
	if !contains(allTargetTypes, targetType) {
		return badData("application_method.target_type should be one of %s", strings.Join(allTargetTypes, ", "))
	}
	if !contains(allMethodTypes, methodType) {
		return badData("application_method.type should be one of %s", strings.Join(allMethodTypes, ", "))
	}
	alloc := allocation
	if alloc == nil {
		alloc = ""
	}
	if contains(allowedAllocationTargetTypes, targetType) && !contains(allowedAllocationTypes, alloc) {
		return badData("application_method.allocation should be either '%s' when application_method.target_type is either '%s'",
			strings.Join(allowedAllocationTypes, " OR "), strings.Join(allowedAllocationTargetTypes, " OR "))
	}
	if jsTruthy(allocation) && !contains(allAllocationTypes, allocation) {
		return badData("application_method.allocation should be one of %s", strings.Join(allAllocationTypes, ", "))
	}
	if jsTruthy(allocation) && contains(allowedAllocationForQuantity, allocation) && !isPresent(maxQuantity) {
		return badData("application_method.max_quantity is required when application_method.allocation is '%s'", strings.Join(allowedAllocationForQuantity, " OR "))
	}
	if allocation == "once" && targetType == "order" {
		return badData("application_method.allocation 'once' is not compatible with target_type 'order'")
	}
	return nil
}

func jsNum(v any) string {
	if f, ok := v.(float64); ok {
		return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%f", f), "0"), ".")
	}
	return str(v)
}

// validatePromotionRuleAttributes mirrors validatePromotionRuleAttributes.
func validatePromotionRuleAttributes(rules []*dal.Record) error {
	var errs []string
	for _, r := range rules {
		if !isPresent(get(r, "attribute")) {
			errs = append(errs, "rules[].attribute is a required field")
		}
		op := get(r, "operator")
		if !isPresent(op) {
			errs = append(errs, "rules[].operator is a required field")
		}
		if isPresent(op) {
			if !contains(ruleOperators, op) {
				errs = append(errs, fmt.Sprintf("rules[].operator (%s) is invalid. It should be one of %s", str(op), strings.Join(ruleOperators, ", ")))
			}
		} else {
			errs = append(errs, "rules[].operator is a required field")
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return badData("%s", strings.Join(errs, ", "))
}

func recordsOf(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		out := make([]*dal.Record, 0, len(t))
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

// createPromotionRulesAndValues mirrors createPromotionRulesAndValues_:
// relationName is promotions, method_target_rules or method_buy_rules and
// owner the promotion or application method (with promotionType the type
// of the promotion).
func (s *PromotionModuleService) createPromotionRulesAndValues(ctx context.Context, rules []*dal.Record, relationName string, ownerID any, promotionType any) ([]*dal.Record, error) {
	if len(rules) == 0 {
		return []*dal.Record{}, nil
	}
	if relationName == "method_buy_rules" && promotionType == "standard" {
		return nil, badData("Can't add buy rules to a standard promotion")
	}
	if err := validatePromotionRuleAttributes(rules); err != nil {
		return nil, err
	}
	var created []*dal.Record
	var values []*dal.Record
	for _, r := range rules {
		data := copyRecord(r, "values")
		ref := dal.NewRecord()
		ref.Set("id", ownerID)
		data.Set(relationName, []any{ref})
		rows, err := s.Internal("PromotionRule").Create(ctx, []*dal.Record{data})
		if err != nil {
			return nil, err
		}
		rule := rows[0]
		created = append(created, rule)
		vals := get(r, "values")
		list, isList := vals.([]any)
		if !isList {
			list = []any{vals}
		}
		for _, v := range list {
			n := dal.NewRecord()
			n.Set("value", v)
			n.Set("promotion_rule", get(rule, "id"))
			values = append(values, n)
		}
	}
	if len(values) > 0 {
		if _, err := s.Internal("PromotionRuleValue").Create(ctx, values); err != nil {
			return nil, err
		}
	}
	return created, nil
}

var promotionRelations = []string{
	"application_method", "application_method.target_rules", "application_method.target_rules.values",
	"application_method.buy_rules", "application_method.buy_rules.values", "rules", "rules.values", "campaign", "campaign.budget",
}

func (s *PromotionModuleService) listPromotions(ctx context.Context, promotionIDs []any, relations []string) ([]*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("id", promotionIDs)
	return s.ListBySuffix(ctx, "Promotions", f, utilsmodulessdk.FindConfig{Relations: relations})
}

type methodRules struct {
	target, buy []*dal.Record
}

// CreatePromotions mirrors createPromotions.
func (s *PromotionModuleService) CreatePromotions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var created []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := s.with(tx)
		var campaignIDs []any
		for _, d := range data {
			if id, ok := get(d, "campaign_id").(string); ok && id != "" {
				campaignIDs = append(campaignIDs, id)
			}
		}
		var existingCampaigns []*dal.Record
		if len(campaignIDs) > 0 {
			f := dal.NewRecord()
			f.Set("id", campaignIDs)
			var err error
			existingCampaigns, err = tx.Internal("Campaign").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"budget"}})
			if err != nil {
				return err
			}
		}
		findCampaign := func(id any) *dal.Record {
			for _, c := range existingCampaigns {
				if get(c, "id") == id {
					return c
				}
			}
			return nil
		}
		methodByCode := map[string]*dal.Record{}
		rulesByCode := map[string][]*dal.Record{}
		campaignByCode := map[string]*dal.Record{}
		var promotionsData []*dal.Record
		for _, d := range data {
			code := str(get(d, "code"))
			method, _ := get(d, "application_method").(*dal.Record)
			campaign, _ := get(d, "campaign").(*dal.Record)
			campaignID := get(d, "campaign_id")
			promotion := copyRecord(d, "application_method", "rules", "campaign", "campaign_id")
			methodByCode[code] = method
			if rules := get(d, "rules"); rules != nil {
				rulesByCode[code] = recordsOf(rules)
			}
			if campaign != nil && jsTruthy(campaignID) {
				return badData("Provide either the 'campaign' or 'campaign_id' parameter; both cannot be used simultaneously.")
			}
			if campaign == nil && !jsTruthy(campaignID) {
				promotionsData = append(promotionsData, promotion)
				continue
			}
			existing := findCampaign(campaignID)
			if jsTruthy(campaignID) && existing == nil {
				return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Could not find campaign with id - %v", campaignID))
			}
			budget, _ := get(campaign, "budget").(*dal.Record)
			existingBudget, _ := get(existing, "budget").(*dal.Record)
			currency := or(get(budget, "currency_code"), get(existingBudget, "currency_code"))
			if get(budget, "type") == "spend" && currency != get(method, "currency_code") {
				return badData("Currency between promotion and campaigns should match")
			}
			if campaign != nil {
				campaignByCode[code] = campaign
			}
			promotion.Set("campaign_id", campaignID)
			promotionsData = append(promotionsData, promotion)
		}
		var err error
		created, err = tx.Internal("Promotion").Create(ctx, promotionsData)
		if err != nil {
			return err
		}
		type pendingCampaign struct {
			data      *dal.Record
			promotion any
		}
		var campaigns []pendingCampaign
		var methods []*dal.Record
		rulesByPromotion := map[string]methodRules{}
		for _, promotion := range created {
			code := str(get(promotion, "code"))
			if c := campaignByCode[code]; c != nil {
				campaigns = append(campaigns, pendingCampaign{data: c, promotion: get(promotion, "id")})
			}
			if method := methodByCode[code]; method != nil {
				target := recordsOf(get(method, "target_rules"))
				buy := recordsOf(get(method, "buy_rules"))
				m := copyRecord(method, "target_rules", "buy_rules")
				m.Set("promotion", get(promotion, "id"))
				if get(m, "target_type") == "order" && len(target) > 0 {
					return badData("Target rules for application method with target type (order) is not allowed")
				}
				if get(promotion, "type") == "buyget" && len(buy) == 0 {
					return badData("Buy rules are required for buyget promotion type")
				}
				if get(promotion, "type") == "buyget" && len(target) == 0 {
					return badData("Target rules are required for buyget promotion type")
				}
				if err := validateApplicationMethodAttributes(m, promotion); err != nil {
					return err
				}
				methods = append(methods, m)
				rulesByPromotion[str(get(promotion, "id"))] = methodRules{target: target, buy: buy}
			}
			if rules, ok := rulesByCode[code]; ok {
				if _, err := t.createPromotionRulesAndValues(ctx, rules, "promotions", get(promotion, "id"), get(promotion, "type")); err != nil {
					return err
				}
			}
		}
		var createdMethods []*dal.Record
		if len(methods) > 0 {
			createdMethods, err = tx.Internal("ApplicationMethod").Create(ctx, methods)
			if err != nil {
				return err
			}
		}
		if len(campaigns) > 0 {
			input := make([]*dal.Record, len(campaigns))
			for i, c := range campaigns {
				input[i] = c.data
			}
			createdCampaigns, err := t.CreateCampaigns(ctx, input)
			if err != nil {
				return err
			}
			for _, c := range campaigns {
				for _, cc := range createdCampaigns {
					if get(cc, "campaign_identifier") == get(c.data, "campaign_identifier") {
						u := dal.NewRecord()
						u.Set("id", c.promotion)
						u.Set("campaign_id", get(cc, "id"))
						if _, err := tx.Internal("Promotion").Update(ctx, []any{u}); err != nil {
							return err
						}
						break
					}
				}
			}
		}
		promotionTypes := map[string]any{}
		for _, p := range created {
			promotionTypes[str(get(p, "id"))] = get(p, "type")
		}
		for i, m := range createdMethods {
			promotionID := str(get(methods[i], "promotion"))
			rules := rulesByPromotion[promotionID]
			if len(rules.target) > 0 {
				if _, err := t.createPromotionRulesAndValues(ctx, rules.target, "method_target_rules", get(m, "id"), promotionTypes[promotionID]); err != nil {
					return err
				}
			}
			if len(rules.buy) > 0 {
				if _, err := t.createPromotionRulesAndValues(ctx, rules.buy, "method_buy_rules", get(m, "id"), promotionTypes[promotionID]); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.listPromotions(ctx, ids(created), promotionRelations)
}

// UpdatePromotions mirrors updatePromotions.
func (s *PromotionModuleService) UpdatePromotions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var updatedIDs []any
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		promotionIDs := make([]any, len(data))
		var campaignIDs []any
		for i, d := range data {
			promotionIDs[i] = get(d, "id")
			if c := get(d, "campaign_id"); isPresent(c) {
				campaignIDs = append(campaignIDs, c)
			}
		}
		f := dal.NewRecord()
		f.Set("id", promotionIDs)
		existing, err := tx.Internal("Promotion").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"application_method"}})
		if err != nil {
			return err
		}
		if campaignIDs == nil {
			campaignIDs = []any{}
		}
		cf := dal.NewRecord()
		cf.Set("id", campaignIDs)
		campaigns, err := tx.Internal("Campaign").List(ctx, cf, utilsmodulessdk.FindConfig{Relations: []string{"budget"}})
		if err != nil {
			return err
		}
		byID := map[string]*dal.Record{}
		for _, p := range existing {
			byID[str(get(p, "id"))] = p
		}
		var promotions []any
		var methods []any
		for _, d := range data {
			method, _ := get(d, "application_method").(*dal.Record)
			campaignID, hasCampaignID := d.Get("campaign_id")
			promotion := copyRecord(d, "application_method", "campaign_id")
			var campaign *dal.Record
			for _, c := range campaigns {
				if get(c, "id") == campaignID {
					campaign = c
				}
			}
			ex := byID[str(get(promotion, "id"))]
			exMethod, _ := get(ex, "application_method").(*dal.Record)
			currency := or(get(exMethod, "currency_code"), get(method, "currency_code"))
			if limit := get(promotion, "limit"); limit != nil {
				used := get(ex, "used")
				if used == nil {
					used = float64(0)
				}
				if totals.Lt(limit, used) {
					return badData("Promotion limit (%s) cannot be less than current usage (%s)", jsNum(limit), jsNum(used))
				}
			}
			if jsTruthy(campaignID) && campaign == nil {
				return badData("Could not find campaign with id %v", campaignID)
			}
			budget, _ := get(campaign, "budget").(*dal.Record)
			if jsTruthy(campaignID) && get(budget, "type") == "spend" && get(budget, "currency_code") != currency {
				return badData("Currency code doesn't match for campaign (%v) and promotion (%v)", campaignID, get(ex, "id"))
			}
			if hasCampaignID {
				promotion.Set("campaign_id", campaignID)
			}
			promotions = append(promotions, promotion)
			if method == nil || exMethod == nil {
				continue
			}
			m := copyRecord(method)
			if a := get(m, "allocation"); jsTruthy(a) && !contains(allowedAllocationForQuantity, a) {
				m.Set("max_quantity", nil)
				exMethod.Set("max_quantity", nil)
			}
			if err := validateApplicationMethodAttributes(m, ex); err != nil {
				return err
			}
			m.Set("id", get(exMethod, "id"))
			methods = append(methods, m)
		}
		res, err := tx.Internal("Promotion").Update(ctx, promotions)
		if err != nil {
			return err
		}
		switch r := res.(type) {
		case []*dal.Record:
			updatedIDs = ids(r)
		case *dal.Record:
			updatedIDs = []any{get(r, "id")}
		}
		if len(methods) > 0 {
			if _, err := tx.Internal("ApplicationMethod").Update(ctx, methods); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.listPromotions(ctx, updatedIDs, []string{"application_method", "application_method.target_rules", "application_method.target_rules.values", "rules", "rules.values", "campaign", "campaign.budget"})
}

func (s *PromotionModuleService) listRules(ctx context.Context, ruleIDs []any) ([]*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("id", ruleIDs)
	return s.ListBySuffix(ctx, "PromotionRules", f, utilsmodulessdk.FindConfig{Relations: []string{"values"}})
}

// UpdatePromotionRules mirrors updatePromotionRules.
func (s *PromotionModuleService) UpdatePromotionRules(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var updated []any
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		ruleIDs := make([]any, len(data))
		for i, d := range data {
			ruleIDs[i] = get(d, "id")
		}
		f := dal.NewRecord()
		f.Set("id", ruleIDs)
		rules, err := tx.Internal("PromotionRule").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"values"}})
		if err != nil {
			return err
		}
		byID := map[string]*dal.Record{}
		for _, r := range rules {
			byID[str(get(r, "id"))] = r
		}
		seen := map[string]bool{}
		var invalid []string
		for _, id := range ruleIDs {
			k := str(id)
			if seen[k] {
				continue
			}
			seen[k] = true
			if byID[k] == nil {
				invalid = append(invalid, k)
			}
		}
		if len(invalid) > 0 {
			return badData("Promotion rules with id - %s not found", strings.Join(invalid, ", "))
		}
		var toUpdate []any
		var valueIDsToDelete []any
		var valuesToCreate []*dal.Record
		for _, d := range data {
			toUpdate = append(toUpdate, copyRecord(d, "values"))
			values, has := d.Get("values")
			if !has || values == nil {
				continue
			}
			rule := byID[str(get(d, "id"))]
			for _, v := range recordsOf(get(rule, "values")) {
				valueIDsToDelete = append(valueIDsToDelete, get(v, "id"))
			}
			list, isList := values.([]any)
			if !isList {
				list = []any{values}
			}
			for _, v := range list {
				n := dal.NewRecord()
				n.Set("value", v)
				n.Set("promotion_rule", get(rule, "id"))
				valuesToCreate = append(valuesToCreate, n)
			}
		}
		res, err := tx.Internal("PromotionRule").Update(ctx, toUpdate)
		if err != nil {
			return err
		}
		if len(valueIDsToDelete) > 0 {
			if _, err := tx.Internal("PromotionRuleValue").Delete(ctx, valueIDsToDelete); err != nil {
				return err
			}
		}
		if len(valuesToCreate) > 0 {
			if _, err := tx.Internal("PromotionRuleValue").Create(ctx, valuesToCreate); err != nil {
				return err
			}
		}
		switch r := res.(type) {
		case []*dal.Record:
			updated = ids(r)
		case *dal.Record:
			updated = []any{get(r, "id")}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.listRules(ctx, updated)
}

func (s *PromotionModuleService) addRules(ctx context.Context, promotionID any, rules []*dal.Record, relationName string) ([]*dal.Record, error) {
	var created []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := s.with(tx)
		config := utilsmodulessdk.FindConfig{}
		if relationName != "promotions" {
			config.Relations = []string{"application_method"}
		}
		promotion, err := tx.Internal("Promotion").Retrieve(ctx, promotionID, config)
		if err != nil {
			return err
		}
		ownerID := get(promotion, "id")
		if relationName != "promotions" {
			am, _ := get(promotion, "application_method").(*dal.Record)
			if am == nil {
				return badData("application_method for promotion not found")
			}
			ownerID = get(am, "id")
		}
		created, err = t.createPromotionRulesAndValues(ctx, rules, relationName, ownerID, get(promotion, "type"))
		return err
	})
	if err != nil {
		return nil, err
	}
	return s.listRules(ctx, ids(created))
}

// AddPromotionRules mirrors addPromotionRules.
func (s *PromotionModuleService) AddPromotionRules(ctx context.Context, promotionID any, rules []*dal.Record) ([]*dal.Record, error) {
	return s.addRules(ctx, promotionID, rules, "promotions")
}

// AddPromotionTargetRules mirrors addPromotionTargetRules.
func (s *PromotionModuleService) AddPromotionTargetRules(ctx context.Context, promotionID any, rules []*dal.Record) ([]*dal.Record, error) {
	return s.addRules(ctx, promotionID, rules, "method_target_rules")
}

// AddPromotionBuyRules mirrors addPromotionBuyRules.
func (s *PromotionModuleService) AddPromotionBuyRules(ctx context.Context, promotionID any, rules []*dal.Record) ([]*dal.Record, error) {
	return s.addRules(ctx, promotionID, rules, "method_buy_rules")
}

// RemovePromotionRules mirrors removePromotionRules.
func (s *PromotionModuleService) RemovePromotionRules(ctx context.Context, promotionID any, ruleIDs []any) error {
	return s.removeRules(ctx, promotionID, ruleIDs, "rules")
}

// RemovePromotionTargetRules mirrors removePromotionTargetRules.
func (s *PromotionModuleService) RemovePromotionTargetRules(ctx context.Context, promotionID any, ruleIDs []any) error {
	return s.removeRules(ctx, promotionID, ruleIDs, "target_rules")
}

// RemovePromotionBuyRules mirrors removePromotionBuyRules.
func (s *PromotionModuleService) RemovePromotionBuyRules(ctx context.Context, promotionID any, ruleIDs []any) error {
	// no @EmitEvents upstream
	return s.removeRules(dal.WithoutEmit(ctx), promotionID, ruleIDs, "buy_rules")
}

func (s *PromotionModuleService) removeRules(ctx context.Context, promotionID any, ruleIDs []any, relation string) error {
	return s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		rel := "rules"
		if relation != "rules" {
			rel = "application_method." + relation
		}
		promotion, err := tx.Internal("Promotion").Retrieve(ctx, promotionID, utilsmodulessdk.FindConfig{Relations: []string{rel}})
		if err != nil {
			return err
		}
		var existing []*dal.Record
		if relation == "rules" {
			existing = recordsOf(get(promotion, "rules"))
		} else {
			am, _ := get(promotion, "application_method").(*dal.Record)
			if am == nil {
				return badData("application_method for promotion not found")
			}
			existing = recordsOf(get(am, relation))
		}
		wanted := map[string]bool{}
		for _, id := range ruleIDs {
			wanted[str(id)] = true
		}
		var toRemove []any
		if relation == "rules" {
			have := map[string]bool{}
			for _, r := range existing {
				have[str(get(r, "id"))] = true
			}
			for _, id := range ruleIDs {
				if have[str(id)] {
					toRemove = append(toRemove, id)
				}
			}
		} else {
			for _, r := range existing {
				if wanted[str(get(r, "id"))] {
					toRemove = append(toRemove, get(r, "id"))
				}
			}
		}
		if len(toRemove) == 0 {
			return nil
		}
		_, err = tx.Internal("PromotionRule").Delete(ctx, toRemove)
		return err
	})
}
