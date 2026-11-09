package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func ruleIDs(rows []*dal.Record) []any {
	out := make([]any, len(rows))
	for i, r := range rows {
		out[i], _ = r.Get("id")
	}
	return out
}

// CreateShippingOptionRulesStep mirrors createShippingOptionRulesStep.
var CreateShippingOptionRulesStep = composer.CreateStep("create-shipping-option-rules",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := Fulfillment(ctx.Container).CreateShippingOptionRules(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(ruleIDs(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := Fulfillment(ctx.Container).Internal("ShippingOptionRule").Delete(ctx.Context, ids)
		return err
	})

// UpdateShippingOptionRulesStep mirrors updateShippingOptionRulesStep.
var UpdateShippingOptionRulesStep = composer.CreateStep("update-shipping-option-rules",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(data) == 0 {
			return composer.NewStepResponse[[]*dal.Record](nil), nil
		}
		svc := Fulfillment(ctx.Container)
		ids := ruleIDs(data)
		f := dal.NewRecord()
		f.Set("id", ids)
		before, err := svc.Internal("ShippingOptionRule").List(ctx.Context, f, utilsmodulessdk.FindConfig{Select: []string{"id", "attribute", "operator", "value", "shipping_option_id"}})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		updated, err := svc.UpdateShippingOptionRules(ctx.Context, data)
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
		_, err := Fulfillment(ctx.Container).UpdateShippingOptionRules(ctx.Context, before)
		return err
	})

// DeleteShippingOptionRulesStep mirrors deleteShippingOptionRulesStep.
var DeleteShippingOptionRulesStep = composer.CreateStep("delete-shipping-option-rules",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[[]any], error) {
		if len(ids) == 0 {
			return composer.NewStepResponse[[]any](nil), nil
		}
		svc := Fulfillment(ctx.Container)
		f := dal.NewRecord()
		f.Set("id", ids)
		before, err := svc.Internal("ShippingOptionRule").List(ctx.Context, f, utilsmodulessdk.FindConfig{Select: []string{"attribute", "operator", "value", "shipping_option_id"}})
		if err != nil {
			return composer.StepResponse[[]any]{}, err
		}
		if _, err := svc.Internal("ShippingOptionRule").Delete(ctx.Context, ids); err != nil {
			return composer.StepResponse[[]any]{}, err
		}
		return composer.NewStepResponse(ids, any(before)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		before, _ := data.([]*dal.Record)
		if len(before) == 0 {
			return nil
		}
		restore := make([]*dal.Record, len(before))
		for i, r := range before {
			n := dal.NewRecord()
			for _, k := range []string{"id", "attribute", "operator", "value", "shipping_option_id"} {
				v, _ := r.Get(k)
				n.Set(k, v)
			}
			restore[i] = n
		}
		_, err := Fulfillment(ctx.Container).CreateShippingOptionRules(ctx.Context, restore)
		return err
	})
