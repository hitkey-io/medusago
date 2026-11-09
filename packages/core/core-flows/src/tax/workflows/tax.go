// Package workflows mirrors @medusajs/core-flows/src/tax/workflows.
package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/tax/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateTaxRegionsWorkflow mirrors createTaxRegionsWorkflow.
var CreateTaxRegionsWorkflow = composer.CreateWorkflow("create-tax-regions",
	func(e *composer.Execution, input []*dal.Record) ([]*dal.Record, error) {
		return steps.CreateTaxRegionsStep.Run(e, input)
	})

// UpdateTaxRegionsWorkflow mirrors updateTaxRegionsWorkflow.
var UpdateTaxRegionsWorkflow = composer.CreateWorkflow("update-tax-regions",
	func(e *composer.Execution, input []*dal.Record) ([]*dal.Record, error) {
		return steps.UpdateTaxRegionsStep.Run(e, input)
	})

// DeleteTaxRegionsWorkflow mirrors deleteTaxRegionsWorkflow.
var DeleteTaxRegionsWorkflow = composer.CreateWorkflow("delete-tax-regions",
	func(e *composer.Execution, ids []any) (any, error) {
		return steps.DeleteTaxRegionsStep.Run(e, ids)
	})

// CreateTaxRatesWorkflow mirrors createTaxRatesWorkflow.
var CreateTaxRatesWorkflow = composer.CreateWorkflow("create-tax-rates",
	func(e *composer.Execution, input []*dal.Record) ([]*dal.Record, error) {
		return steps.CreateTaxRatesStep.Run(e, input)
	})

// DeleteTaxRatesWorkflow mirrors deleteTaxRatesWorkflow.
var DeleteTaxRatesWorkflow = composer.CreateWorkflow("delete-tax-rates",
	func(e *composer.Execution, ids []any) (any, error) {
		return steps.DeleteTaxRatesStep.Run(e, ids)
	})

// CreateTaxRateRulesWorkflow mirrors createTaxRateRulesWorkflow.
var CreateTaxRateRulesWorkflow = composer.CreateWorkflow("create-tax-rate-rules",
	func(e *composer.Execution, rules []*dal.Record) ([]*dal.Record, error) {
		return steps.CreateTaxRateRulesStep.Run(e, rules)
	})

// DeleteTaxRateRulesWorkflow mirrors deleteTaxRateRulesWorkflow.
var DeleteTaxRateRulesWorkflow = composer.CreateWorkflow("delete-tax-rate-rules",
	func(e *composer.Execution, ids []any) (any, error) {
		return steps.DeleteTaxRateRulesStep.Run(e, ids)
	})

// SetTaxRateRulesInput mirrors SetTaxRatesRulesWorkflowInput.
type SetTaxRateRulesInput struct {
	TaxRateIDs []any
	Rules      []*dal.Record
}

func rulesFor(rules []*dal.Record, rateIDs []any, createdBy any, withCreatedBy bool) []*dal.Record {
	var out []*dal.Record
	for _, r := range rules {
		for _, id := range rateIDs {
			n := dal.NewRecord()
			for _, k := range r.Keys() {
				v, _ := r.Get(k)
				n.Set(k, v)
			}
			if withCreatedBy {
				n.Set("created_by", createdBy)
			}
			n.Set("tax_rate_id", id)
			out = append(out, n)
		}
	}
	if out == nil {
		out = []*dal.Record{}
	}
	return out
}

// SetTaxRateRulesWorkflow mirrors setTaxRateRulesWorkflow.
var SetTaxRateRulesWorkflow = composer.CreateWorkflow("set-tax-rate-rules",
	func(e *composer.Execution, input SetTaxRateRulesInput) ([]*dal.Record, error) {
		sel := dal.NewRecord()
		sel.Set("tax_rate_id", input.TaxRateIDs)
		ruleIDs, err := steps.ListTaxRateRuleIdsStep.Run(e, sel)
		if err != nil {
			return nil, err
		}
		if _, err := steps.DeleteTaxRateRulesStep.Run(e, ruleIDs); err != nil {
			return nil, err
		}
		return steps.CreateTaxRateRulesStep.Run(e, rulesFor(input.Rules, input.TaxRateIDs, nil, false))
	})

// UpdateTaxRatesInput mirrors UpdateTaxRatesWorkflowInput.
type UpdateTaxRatesInput struct {
	Selector *dal.Record
	Update   *dal.Record
}

type maybeListInput struct {
	TaxRateIDs []any
	HasRules   bool
}

// maybeListTaxRateRuleIdsStep mirrors maybeListTaxRateRuleIdsStep.
var maybeListTaxRateRuleIdsStep = composer.CreateStep("maybe-list-tax-rate-rule-ids",
	func(ctx composer.StepExecutionContext, in maybeListInput) (composer.StepResponse[[]any], error) {
		if !in.HasRules {
			return composer.NewStepResponse([]any{}), nil
		}
		out, err := steps.ListTaxRateRuleIDs(ctx, in.TaxRateIDs)
		if err != nil {
			return composer.StepResponse[[]any]{}, err
		}
		return composer.NewStepResponse(out), nil
	}, nil)

// UpdateTaxRatesWorkflow mirrors updateTaxRatesWorkflow.
var UpdateTaxRatesWorkflow = composer.CreateWorkflow("update-tax-rates",
	func(e *composer.Execution, input UpdateTaxRatesInput) ([]*dal.Record, error) {
		rulesV, hasRules := input.Update.Get("rules")
		hasRules = hasRules && rulesV != nil
		clean := dal.NewRecord()
		for _, k := range input.Update.Keys() {
			if k == "rules" && hasRules {
				continue
			}
			v, _ := input.Update.Get(k)
			clean.Set(k, v)
		}
		updated, err := steps.UpdateTaxRatesStep.Run(e, steps.UpdateTaxRatesStepInput{Selector: input.Selector, Update: clean})
		if err != nil {
			return nil, err
		}
		rateIDs := make([]any, len(updated))
		for i, r := range updated {
			rateIDs[i], _ = r.Get("id")
		}
		ruleIDs, err := maybeListTaxRateRuleIdsStep.Run(e, maybeListInput{TaxRateIDs: rateIDs, HasRules: hasRules})
		if err != nil {
			return nil, err
		}
		if _, err := steps.DeleteTaxRateRulesStep.Run(e, ruleIDs); err != nil {
			return nil, err
		}
		var toCreate []*dal.Record
		if hasRules {
			var rules []*dal.Record
			switch t := rulesV.(type) {
			case []any:
				for _, x := range t {
					if r, ok := x.(*dal.Record); ok {
						rules = append(rules, r)
					}
				}
			case []*dal.Record:
				rules = t
			}
			updatedBy, _ := input.Update.Get("updated_by")
			toCreate = rulesFor(rules, rateIDs, updatedBy, true)
		} else {
			toCreate = []*dal.Record{}
		}
		if _, err := steps.CreateTaxRateRulesStep.Run(e, toCreate); err != nil {
			return nil, err
		}
		return updated, nil
	})
