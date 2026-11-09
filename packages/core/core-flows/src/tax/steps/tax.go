// Package steps mirrors @medusajs/core-flows/src/tax/steps.
package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/modules/tax/src/services"
)

// TaxService resolves the tax module service.
func TaxService(c composer.Container) *services.TaxModuleService {
	s, _ := c.Resolve(utilsmodulessdk.TAX).(*services.TaxModuleService)
	return s
}

func ids(rs []*dal.Record) []any {
	out := make([]any, len(rs))
	for i, r := range rs {
		out[i], _ = r.Get("id")
	}
	return out
}

func createStep(name, className string, create func(ctx composer.StepExecutionContext, data []*dal.Record) ([]*dal.Record, error)) *composer.Step[[]*dal.Record, []*dal.Record] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
			created, err := create(ctx, data)
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
			return composer.NewStepResponse(created, any(ids(created))), nil
		},
		func(ctx composer.StepExecutionContext, data any) error {
			list, _ := data.([]any)
			if len(list) == 0 {
				return nil
			}
			_, err := TaxService(ctx.Container).Internal(className).Delete(ctx.Context, list)
			return err
		})
}

func softDeleteStep(name, className string, skipEmpty bool) *composer.Step[[]any, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, list []any) (composer.StepResponse[any], error) {
			if skipEmpty && len(list) == 0 {
				return composer.NewStepResponse[any](nil, any([]any{})), nil
			}
			if _, err := TaxService(ctx.Container).Internal(className).SoftDelete(ctx.Context, list); err != nil {
				return composer.StepResponse[any]{}, err
			}
			return composer.NewStepResponse[any](nil, any(list)), nil
		},
		func(ctx composer.StepExecutionContext, data any) error {
			list, _ := data.([]any)
			if len(list) == 0 {
				return nil
			}
			_, err := TaxService(ctx.Container).Internal(className).Restore(ctx.Context, list)
			return err
		})
}

// CreateTaxRegionsStep mirrors createTaxRegionsStep.
var CreateTaxRegionsStep = createStep("create-tax-regions", "TaxRegion", func(ctx composer.StepExecutionContext, data []*dal.Record) ([]*dal.Record, error) {
	return TaxService(ctx.Container).CreateTaxRegions(ctx.Context, data)
})

// CreateTaxRatesStep mirrors createTaxRatesStep.
var CreateTaxRatesStep = createStep("create-tax-rates", "TaxRate", func(ctx composer.StepExecutionContext, data []*dal.Record) ([]*dal.Record, error) {
	return TaxService(ctx.Container).CreateTaxRates(ctx.Context, data)
})

// CreateTaxRateRulesStep mirrors createTaxRateRulesStep.
var CreateTaxRateRulesStep = createStep("create-tax-rate-rules", "TaxRateRule", func(ctx composer.StepExecutionContext, data []*dal.Record) ([]*dal.Record, error) {
	return TaxService(ctx.Container).CreateTaxRateRules(ctx.Context, data)
})

// DeleteTaxRegionsStep mirrors deleteTaxRegionsStep.
var DeleteTaxRegionsStep = softDeleteStep("delete-tax-regions", "TaxRegion", false)

// DeleteTaxRatesStep mirrors deleteTaxRatesStep.
var DeleteTaxRatesStep = softDeleteStep("delete-tax-rates", "TaxRate", false)

// DeleteTaxRateRulesStep mirrors deleteTaxRateRulesStep.
var DeleteTaxRateRulesStep = softDeleteStep("delete-tax-rate-rules", "TaxRateRule", true)

// UpdateTaxRatesStepInput mirrors UpdateTaxRatesStepInput.
type UpdateTaxRatesStepInput struct {
	Selector *dal.Record
	Update   *dal.Record
}

// UpdateTaxRatesStep mirrors updateTaxRatesStep.
var UpdateTaxRatesStep = composer.CreateStep("update-tax-rates",
	func(ctx composer.StepExecutionContext, in UpdateTaxRatesStepInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := TaxService(ctx.Container)
		selects, relations := common.GetSelectsAndRelationsFromObjectArray([]common.ObjectLike{in.Update}, common.SelectsAndRelationsOptions{}, "")
		prev, err := svc.ListBySuffix(ctx.Context, "TaxRates", in.Selector, utilsmodulessdk.FindConfig{Select: selects, Relations: relations})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		out, err := svc.UpdateTaxRates(ctx.Context, in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(out, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		_, err := TaxService(ctx.Container).UpsertTaxRates(ctx.Context, prev)
		return err
	})

// UpdateTaxRegionsStep mirrors updateTaxRegionsStep.
var UpdateTaxRegionsStep = composer.CreateStep("update-tax-regions",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(data) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
		}
		svc := TaxService(ctx.Container)
		objs := make([]common.ObjectLike, len(data))
		for i, d := range data {
			objs[i] = d
		}
		selects, relations := common.GetSelectsAndRelationsFromObjectArray(objs, common.SelectsAndRelationsOptions{}, "")
		f := dal.NewRecord()
		f.Set("id", ids(data))
		prev, err := svc.ListBySuffix(ctx.Context, "TaxRegions", f, utilsmodulessdk.FindConfig{Select: selects, Relations: relations})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		update := make([]*dal.Record, len(data))
		for i, d := range data {
			update[i] = pick(d, "id", "province_code", "metadata", "provider_id")
		}
		out, err := svc.UpdateTaxRegions(ctx.Context, update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(out, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		update := make([]*dal.Record, len(prev))
		for i, d := range prev {
			update[i] = pick(d, "id", "province_code", "metadata")
		}
		_, err := TaxService(ctx.Container).UpdateTaxRegions(ctx.Context, update)
		return err
	})

// pick copies the defined keys (removeUndefined).
func pick(r *dal.Record, keys ...string) *dal.Record {
	out := dal.NewRecord()
	for _, k := range keys {
		if v, ok := r.Get(k); ok {
			out.Set(k, v)
		}
	}
	return out
}

// ListTaxRateRuleIDs lists the ids of the rules of the given rates.
func ListTaxRateRuleIDs(ctx composer.StepExecutionContext, rateIDs []any) ([]any, error) {
	f := dal.NewRecord()
	f.Set("tax_rate_id", rateIDs)
	rules, err := TaxService(ctx.Container).ListBySuffix(ctx.Context, "TaxRateRules", f, utilsmodulessdk.FindConfig{Select: []string{"id"}})
	if err != nil {
		return nil, err
	}
	return ids(rules), nil
}

// ListTaxRateRuleIdsStep mirrors listTaxRateRuleIdsStep.
var ListTaxRateRuleIdsStep = composer.CreateStep("list-tax-rate-rule-ids",
	func(ctx composer.StepExecutionContext, selector *dal.Record) (composer.StepResponse[[]any], error) {
		rules, err := TaxService(ctx.Container).ListBySuffix(ctx.Context, "TaxRateRules", selector, utilsmodulessdk.FindConfig{Select: []string{"id"}})
		if err != nil {
			return composer.StepResponse[[]any]{}, err
		}
		return composer.NewStepResponse(ids(rules)), nil
	}, nil)
