package steps

import (
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func records(v any) []*dal.Record {
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

// CreatePromotionsStep mirrors createPromotionsStep.
var CreatePromotionsStep = composer.CreateStep("create-promotions",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := Promotion(ctx.Container).CreatePromotions(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(commonsteps.RecordIDs(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := Promotion(ctx.Container).Internal("Promotion").Delete(ctx.Context, ids)
		return err
	})

type promotionsBefore struct {
	before []*dal.Record
	keys   map[string]bool
}

// UpdatePromotionsStep mirrors updatePromotionsStep.
var UpdatePromotionsStep = composer.CreateStep("update-promotions",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(data) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any(promotionsBefore{})), nil
		}
		svc := Promotion(ctx.Container)
		ids := make([]any, len(data))
		keys := map[string]bool{"id": true}
		for i, d := range data {
			ids[i] = get(d, "id")
			for _, k := range d.Keys() {
				keys[k] = true
			}
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		var relations []string
		if keys["application_method"] {
			relations = []string{"application_method"}
		}
		before, err := svc.ListBySuffix(ctx.Context, "Promotions", f, utilsmodulessdk.FindConfig{Relations: relations})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		updated, err := svc.UpdatePromotions(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(updated, any(promotionsBefore{before: before, keys: keys})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		c, ok := data.(promotionsBefore)
		if !ok || len(c.before) == 0 {
			return nil
		}
		restore := make([]*dal.Record, len(c.before))
		for i, b := range c.before {
			r := dal.NewRecord()
			for _, k := range b.Keys() {
				if !c.keys[k] || k == "application_method" {
					continue
				}
				v, _ := b.Get(k)
				r.Set(k, v)
			}
			restore[i] = r
		}
		_, err := Promotion(ctx.Container).UpdatePromotions(ctx.Context, restore)
		return err
	})

// DeletePromotionsStep mirrors deletePromotionsStep.
var DeletePromotionsStep = commonsteps.SoftDeleteRecordsStep("delete-promotions", utilsmodulessdk.PROMOTION, "Promotion")

// UpdatePromotionsValidationStep mirrors updatePromotionsValidationStep.
var UpdatePromotionsValidationStep = composer.CreateStep("update-promotions-validation",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[any], error) {
		for _, d := range data {
			switch get(d, "status") {
			case "draft", "active", "inactive":
			default:
				return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "promotion's status should be one of - draft, active, inactive")
			}
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// RuleType mirrors RuleType.
type RuleType string

// Rule types.
const (
	RuleTypeRules       RuleType = "rules"
	RuleTypeTargetRules RuleType = "target_rules"
	RuleTypeBuyRules    RuleType = "buy_rules"
)

// RulesInput mirrors { rule_type, data: { id, rules | rule_ids } }.
type RulesInput struct {
	RuleType RuleType
	ID       any
	Rules    []*dal.Record
	RuleIDs  []any
}

type addedRules struct {
	id                     any
	promotion, buy, target []any
}

// AddRulesToPromotionsStep mirrors addRulesToPromotionsStep.
var AddRulesToPromotionsStep = composer.CreateStep("add-rules-to-promotions",
	func(ctx composer.StepExecutionContext, in RulesInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := Promotion(ctx.Container)
		var promotionRules, buyRules, targetRules []*dal.Record
		var err error
		if in.RuleType == RuleTypeRules {
			if promotionRules, err = svc.AddPromotionRules(ctx.Context, in.ID, in.Rules); err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
		}
		if in.RuleType == RuleTypeBuyRules {
			if buyRules, err = svc.AddPromotionBuyRules(ctx.Context, in.ID, in.Rules); err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
		}
		if in.RuleType == RuleTypeTargetRules {
			if targetRules, err = svc.AddPromotionTargetRules(ctx.Context, in.ID, in.Rules); err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
		}
		all := append(append(append([]*dal.Record{}, promotionRules...), buyRules...), targetRules...)
		// upstream records the buy rule ids as the target rule ids too
		comp := addedRules{id: in.ID, promotion: commonsteps.RecordIDs(promotionRules), buy: commonsteps.RecordIDs(buyRules), target: commonsteps.RecordIDs(buyRules)}
		return composer.NewStepResponse(all, any(comp)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		c, ok := data.(addedRules)
		if !ok {
			return nil
		}
		svc := Promotion(ctx.Container)
		if len(c.promotion) > 0 {
			if err := svc.RemovePromotionRules(ctx.Context, c.id, c.promotion); err != nil {
				return err
			}
		}
		if len(c.buy) > 0 {
			if err := svc.RemovePromotionBuyRules(ctx.Context, c.id, c.buy); err != nil {
				return err
			}
		}
		if len(c.target) > 0 {
			return svc.RemovePromotionBuyRules(ctx.Context, c.id, c.target)
		}
		return nil
	})

// UpdatePromotionRulesStep mirrors updatePromotionRulesStep.
var UpdatePromotionRulesStep = composer.CreateStep("update-promotion-rules",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(data) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
		}
		svc := Promotion(ctx.Container)
		ids := make([]any, len(data))
		for i, d := range data {
			ids[i] = get(d, "id")
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		before, err := svc.ListBySuffix(ctx.Context, "PromotionRules", f, utilsmodulessdk.FindConfig{Relations: []string{"values"}})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		updated, err := svc.UpdatePromotionRules(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(updated, any(before)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		before, _ := data.([]*dal.Record)
		if len(before) == 0 {
			return nil
		}
		restore := make([]*dal.Record, len(before))
		for i, r := range before {
			n := dal.NewRecord()
			for _, k := range []string{"id", "description", "attribute", "operator"} {
				n.Set(k, get(r, k))
			}
			var values []any
			for _, v := range records(get(r, "values")) {
				values = append(values, get(v, "value"))
			}
			n.Set("values", values)
			restore[i] = n
		}
		_, err := Promotion(ctx.Container).UpdatePromotionRules(ctx.Context, restore)
		return err
	})

type removedRules struct {
	id                     any
	promotion, buy, target []*dal.Record
}

func ruleAttributes(rules []*dal.Record) []*dal.Record {
	out := make([]*dal.Record, len(rules))
	for i, r := range rules {
		n := dal.NewRecord()
		n.Set("description", get(r, "description"))
		n.Set("attribute", get(r, "attribute"))
		n.Set("operator", get(r, "operator"))
		var values []any
		for _, v := range records(get(r, "values")) {
			values = append(values, get(v, "value"))
		}
		n.Set("values", values)
		out[i] = n
	}
	return out
}

// RemoveRulesFromPromotionsStep mirrors removeRulesFromPromotionsStep.
var RemoveRulesFromPromotionsStep = composer.CreateStep("remove-rules-from-promotions",
	func(ctx composer.StepExecutionContext, in RulesInput) (composer.StepResponse[any], error) {
		svc := Promotion(ctx.Container)
		promotion, err := svc.Internal("Promotion").Retrieve(ctx.Context, in.ID, utilsmodulessdk.FindConfig{Relations: []string{"rules.values", "application_method.target_rules.values", "application_method.buy_rules.values"}})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		am, _ := get(promotion, "application_method").(*dal.Record)
		comp := removedRules{id: in.ID}
		switch in.RuleType {
		case RuleTypeRules:
			comp.promotion = ruleAttributes(records(get(promotion, "rules")))
			err = svc.RemovePromotionRules(ctx.Context, in.ID, in.RuleIDs)
		case RuleTypeBuyRules:
			if am == nil {
				return composer.StepResponse[any]{}, fmt.Errorf("Cannot read properties of undefined (reading 'map')")
			}
			comp.buy = ruleAttributes(records(get(am, "buy_rules")))
			err = svc.RemovePromotionBuyRules(ctx.Context, in.ID, in.RuleIDs)
		case RuleTypeTargetRules:
			if am == nil {
				return composer.StepResponse[any]{}, fmt.Errorf("Cannot read properties of undefined (reading 'map')")
			}
			comp.target = ruleAttributes(records(get(am, "target_rules")))
			err = svc.RemovePromotionTargetRules(ctx.Context, in.ID, in.RuleIDs)
		}
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(comp)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		c, ok := data.(removedRules)
		if !ok {
			return nil
		}
		svc := Promotion(ctx.Container)
		if len(c.promotion) > 0 {
			if _, err := svc.AddPromotionRules(ctx.Context, c.id, c.promotion); err != nil {
				return err
			}
		}
		if len(c.buy) > 0 {
			if _, err := svc.AddPromotionBuyRules(ctx.Context, c.id, c.buy); err != nil {
				return err
			}
		}
		if len(c.target) > 0 {
			_, err := svc.AddPromotionBuyRules(ctx.Context, c.id, c.target)
			return err
		}
		return nil
	})
