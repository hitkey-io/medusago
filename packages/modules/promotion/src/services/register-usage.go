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

// UsageComputedAction mirrors UsageComputedActions.
type UsageComputedAction struct {
	Amount any
	Code   string
}

// UsageActions converts { amount, code } records.
func UsageActions(list []*dal.Record) []UsageComputedAction {
	out := make([]UsageComputedAction, len(list))
	for i, r := range list {
		code, _ := get(r, "code").(string)
		out[i] = UsageComputedAction{Amount: get(r, "amount"), Code: code}
	}
	return out
}

func usageCodes(actions []UsageComputedAction) []any {
	var out []any
	for _, a := range actions {
		if a.Code != "" {
			out = append(out, a.Code)
		}
	}
	return out
}

func budgetOf(promotion *dal.Record) *dal.Record {
	campaign, _ := get(promotion, "campaign").(*dal.Record)
	b, _ := get(campaign, "budget").(*dal.Record)
	return b
}

func numberLimit(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	}
	return 0, false
}

func orZero(v any) any {
	if v == nil {
		return 0
	}
	return v
}

func lockRows(ctx context.Context, db dal.Querier, table string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	ph := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		ph[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	rows, err := db.Query(ctx, fmt.Sprintf(`select "id" from %q where "id" in (%s) order by "id" asc for update`, table, strings.Join(ph, ", ")), args...)
	if err != nil {
		return err
	}
	rows.Close()
	return rows.Err()
}

// RegisterUsage mirrors registerUsage.
func (s *PromotionModuleService) RegisterUsage(ctx context.Context, actions []*dal.Record, registrationContext *dal.Record) error {
	computedActions := UsageActions(actions)
	return s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := s.with(tx)
		filters := dal.NewRecord()
		filters.Set("code", usageCodes(computedActions))
		config := utilsmodulessdk.FindConfig{Relations: []string{"campaign", "campaign.budget", "campaign.budget.usages"}}
		existing, err := t.listActivePromotions(ctx, filters, config)
		if err != nil {
			return err
		}
		var lockPromotionIDs, lockBudgetIDs []string
		seenBudget := map[string]bool{}
		for _, p := range existing {
			if _, ok := numberLimit(get(p, "limit")); ok {
				lockPromotionIDs = append(lockPromotionIDs, fmt.Sprint(get(p, "id")))
			}
			if b := budgetOf(p); b != nil {
				id := fmt.Sprint(get(b, "id"))
				if !seenBudget[id] {
					seenBudget[id] = true
					lockBudgetIDs = append(lockBudgetIDs, id)
				}
			}
		}
		db := tx.DB()
		if _, err := db.Exec(ctx, "SET LOCAL lock_timeout = '3s'"); err != nil {
			return err
		}
		if err := lockRows(ctx, db, "promotion", lockPromotionIDs); err != nil {
			return err
		}
		if err := lockRows(ctx, db, "promotion_campaign_budget", lockBudgetIDs); err != nil {
			return err
		}
		existing, err = t.listActivePromotions(ctx, filters, config)
		if err != nil {
			return err
		}

		type budgetData struct {
			id   any
			used any
		}
		var budgetOrder []string
		budgets := map[string]*budgetData{}
		for _, p := range existing {
			if b := budgetOf(p); b != nil {
				id := fmt.Sprint(get(b, "id"))
				if _, ok := budgets[id]; !ok {
					budgetOrder = append(budgetOrder, id)
				}
				budgets[id] = &budgetData{id: get(b, "id"), used: get(b, "used")}
			}
		}
		byCode := map[string]*dal.Record{}
		for _, p := range existing {
			byCode[fmt.Sprint(get(p, "code"))] = p
		}
		codeUsed := map[string]bool{}
		var promotionOrder []string
		promotionUsed := map[string]float64{}
		type spendUsage struct {
			attributeValue string
			amount         any
			limit          any
		}
		var spendOrder []string
		spendByAttribute := map[string]*spendUsage{}

		for _, action := range computedActions {
			promotion := byCode[action.Code]
			if promotion == nil {
				continue
			}
			pid := fmt.Sprint(get(promotion, "id"))
			if limit, ok := numberLimit(get(promotion, "limit")); ok {
				used, _ := numberLimit(get(promotion, "used"))
				newUsed := used + 1
				if newUsed > limit {
					return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Promotion usage exceeds the limit.")
				}
				if _, ok := promotionUsed[pid]; !ok {
					promotionOrder = append(promotionOrder, pid)
				}
				promotionUsed[pid] = newUsed
			}
			budget := budgetOf(promotion)
			if budget == nil {
				continue
			}
			bid := fmt.Sprint(get(budget, "id"))
			limit := get(budget, "limit")
			hasLimit := limit != nil && !totals.Eq(limit, 0)
			code := fmt.Sprint(get(promotion, "code"))
			switch get(budget, "type") {
			case "spend":
				data := budgets[bid]
				if data == nil {
					continue
				}
				newUsed := totals.Add(orZero(data.used), action.Amount)
				if hasLimit && totals.Gt(newUsed, limit) {
					return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Promotion usage exceeds the budget limit.")
				}
				data.used = newUsed
			case "usage":
				if codeUsed[code] {
					continue
				}
				newUsed := totals.Add(orZero(get(budget, "used")), 1)
				if hasLimit && totals.Gt(newUsed, limit) {
					return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Promotion usage exceeds the budget limit.")
				}
				budgets[bid] = &budgetData{id: get(budget, "id"), used: newUsed}
				codeUsed[code] = true
			case "use_by_attribute":
				if codeUsed[code] {
					continue
				}
				attributeValue, _ := get(registrationContext, fmt.Sprint(get(budget, "attribute"))).(string)
				if attributeValue == "" {
					continue
				}
				if err := t.registerCampaignBudgetUsageByAttribute(ctx, get(budget, "id"), attributeValue, 1, limit); err != nil {
					return err
				}
				budgets[bid] = &budgetData{id: get(budget, "id"), used: totals.Add(orZero(get(budget, "used")), 1)}
				codeUsed[code] = true
			case "spend_by_attribute":
				data := budgets[bid]
				attributeValue, _ := get(registrationContext, fmt.Sprint(get(budget, "attribute"))).(string)
				if data == nil || attributeValue == "" {
					continue
				}
				prev := spendByAttribute[bid]
				var prevAmount any = 0
				if prev != nil {
					prevAmount = prev.amount
				} else {
					spendOrder = append(spendOrder, bid)
				}
				spendByAttribute[bid] = &spendUsage{attributeValue: attributeValue, amount: totals.Add(prevAmount, action.Amount), limit: limit}
				data.used = totals.Add(orZero(data.used), action.Amount)
			}
		}
		for _, bid := range spendOrder {
			u := spendByAttribute[bid]
			if err := t.registerCampaignBudgetUsageByAttribute(ctx, budgets[bid].id, u.attributeValue, u.amount, u.limit); err != nil {
				return err
			}
		}
		if len(promotionOrder) > 0 {
			var updates []any
			for _, pid := range promotionOrder {
				u := dal.NewRecord()
				u.Set("id", pid)
				u.Set("used", promotionUsed[pid])
				updates = append(updates, u)
			}
			if _, err := tx.Internal("Promotion").Update(ctx, updates); err != nil {
				return err
			}
		}
		if len(budgetOrder) > 0 {
			var updates []any
			for _, bid := range budgetOrder {
				b := budgets[bid]
				u := dal.NewRecord()
				u.Set("id", b.id)
				u.Set("used", b.used)
				updates = append(updates, u)
			}
			if _, err := tx.Internal("CampaignBudget").Update(ctx, updates); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *PromotionModuleService) registerCampaignBudgetUsageByAttribute(ctx context.Context, budgetID any, attributeValue string, amount, limit any) error {
	f := dal.NewRecord()
	f.Set("budget_id", budgetID)
	f.Set("attribute_value", attributeValue)
	rows, err := s.Internal("CampaignBudgetUsage").List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil {
		return err
	}
	var current any = 0
	if len(rows) > 0 {
		current = orZero(get(rows[0], "used"))
	}
	newUsed := totals.Add(current, amount)
	if limit != nil && !totals.Eq(limit, 0) && totals.Gt(newUsed, limit) {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Promotion usage exceeds the budget limit.")
	}
	if len(rows) == 0 {
		c := dal.NewRecord()
		c.Set("budget_id", budgetID)
		c.Set("attribute_value", attributeValue)
		c.Set("used", newUsed)
		_, err := s.Internal("CampaignBudgetUsage").Create(ctx, []*dal.Record{c})
		return err
	}
	u := dal.NewRecord()
	u.Set("id", get(rows[0], "id"))
	u.Set("used", newUsed)
	_, err = s.Internal("CampaignBudgetUsage").Update(ctx, []any{u})
	return err
}

// RevertUsage mirrors revertUsage.
func (s *PromotionModuleService) RevertUsage(ctx context.Context, actions []*dal.Record, registrationContext *dal.Record) error {
	// @EmitEvents runs inside the transaction upstream (before the flush):
	// the mutations wait in the shared aggregator for the next @EmitEvents
	ctx = dal.WithDeferredEmit(ctx)
	computedActions := UsageActions(actions)
	return s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := s.with(tx)
		filters := dal.NewRecord()
		filters.Set("code", usageCodes(computedActions))
		existing, err := t.listActivePromotions(ctx, filters, utilsmodulessdk.FindConfig{Relations: []string{"campaign", "campaign.budget"}})
		if err != nil {
			return err
		}
		type budgetData struct {
			id   any
			used any
		}
		var budgetOrder []string
		budgets := map[string]*budgetData{}
		for _, p := range existing {
			if b := budgetOf(p); b != nil {
				id := fmt.Sprint(get(b, "id"))
				if _, ok := budgets[id]; !ok {
					budgetOrder = append(budgetOrder, id)
				}
				budgets[id] = &budgetData{id: get(b, "id"), used: get(b, "used")}
			}
		}
		byCode := map[string]*dal.Record{}
		for _, p := range existing {
			byCode[fmt.Sprint(get(p, "code"))] = p
		}
		nonNegative := func(v *totals.BN) any {
			if totals.Lt(v, 0) {
				return 0
			}
			return v
		}
		codeUsed := map[string]bool{}
		var promotionOrder []string
		promotionUsed := map[string]float64{}
		type spendUsage struct {
			attributeValue string
			amount         any
		}
		var spendOrder []string
		spendByAttribute := map[string]*spendUsage{}
		for _, action := range computedActions {
			promotion := byCode[action.Code]
			if promotion == nil {
				continue
			}
			pid := fmt.Sprint(get(promotion, "id"))
			if _, ok := numberLimit(get(promotion, "limit")); ok {
				used, _ := numberLimit(get(promotion, "used"))
				newUsed := used - 1
				if newUsed < 0 {
					newUsed = 0
				}
				if _, ok := promotionUsed[pid]; !ok {
					promotionOrder = append(promotionOrder, pid)
				}
				promotionUsed[pid] = newUsed
			}
			budget := budgetOf(promotion)
			if budget == nil {
				continue
			}
			bid := fmt.Sprint(get(budget, "id"))
			code := fmt.Sprint(get(promotion, "code"))
			switch get(budget, "type") {
			case "spend":
				data := budgets[bid]
				if data == nil {
					continue
				}
				data.used = nonNegative(totals.Sub(orZero(data.used), action.Amount))
			case "usage":
				if codeUsed[code] {
					continue
				}
				budgets[bid] = &budgetData{id: get(budget, "id"), used: nonNegative(totals.Sub(orZero(get(budget, "used")), 1))}
				codeUsed[code] = true
			case "use_by_attribute":
				if codeUsed[code] {
					continue
				}
				attributeValue, _ := get(registrationContext, fmt.Sprint(get(budget, "attribute"))).(string)
				if attributeValue == "" {
					continue
				}
				if err := t.revertCampaignBudgetUsageByAttribute(ctx, get(budget, "id"), attributeValue, 1); err != nil {
					return err
				}
				budgets[bid] = &budgetData{id: get(budget, "id"), used: nonNegative(totals.Sub(orZero(get(budget, "used")), 1))}
				codeUsed[code] = true
			case "spend_by_attribute":
				data := budgets[bid]
				attributeValue, _ := get(registrationContext, fmt.Sprint(get(budget, "attribute"))).(string)
				if data == nil || attributeValue == "" {
					continue
				}
				prev := spendByAttribute[bid]
				var prevAmount any = 0
				if prev != nil {
					prevAmount = prev.amount
				} else {
					spendOrder = append(spendOrder, bid)
				}
				spendByAttribute[bid] = &spendUsage{attributeValue: attributeValue, amount: totals.Add(prevAmount, action.Amount)}
				data.used = nonNegative(totals.Sub(orZero(data.used), action.Amount))
			}
		}
		for _, bid := range spendOrder {
			u := spendByAttribute[bid]
			if err := t.revertCampaignBudgetUsageByAttribute(ctx, budgets[bid].id, u.attributeValue, u.amount); err != nil {
				return err
			}
		}
		if len(promotionOrder) > 0 {
			var updates []any
			for _, pid := range promotionOrder {
				u := dal.NewRecord()
				u.Set("id", pid)
				u.Set("used", promotionUsed[pid])
				updates = append(updates, u)
			}
			if _, err := tx.Internal("Promotion").Update(ctx, updates); err != nil {
				return err
			}
		}
		if len(budgetOrder) > 0 {
			var updates []any
			for _, bid := range budgetOrder {
				b := budgets[bid]
				u := dal.NewRecord()
				u.Set("id", b.id)
				u.Set("used", b.used)
				updates = append(updates, u)
			}
			if _, err := tx.Internal("CampaignBudget").Update(ctx, updates); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *PromotionModuleService) revertCampaignBudgetUsageByAttribute(ctx context.Context, budgetID any, attributeValue string, amount any) error {
	f := dal.NewRecord()
	f.Set("budget_id", budgetID)
	f.Set("attribute_value", attributeValue)
	rows, err := s.Internal("CampaignBudgetUsage").List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil || len(rows) == 0 {
		return err
	}
	newUsed := totals.Sub(orZero(get(rows[0], "used")), amount)
	if totals.Lte(newUsed, 0) {
		_, err := s.Internal("CampaignBudgetUsage").Delete(ctx, []any{get(rows[0], "id")})
		return err
	}
	u := dal.NewRecord()
	u.Set("id", get(rows[0], "id"))
	u.Set("used", newUsed)
	_, err = s.Internal("CampaignBudgetUsage").Update(ctx, []any{u})
	return err
}
