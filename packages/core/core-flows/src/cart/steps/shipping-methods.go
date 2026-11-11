package steps

import (
	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ValidateCartShippingOptionsStep mirrors validateCartShippingOptionsStep
// (prefetched options).
var ValidateCartShippingOptionsStep = composer.CreateStep("validate-cart-shipping-options",
	func(ctx composer.StepExecutionContext, in struct {
		OptionIDs []any
		Options   []*dal.Record
	}) (composer.StepResponse[any], error) {
		valid := map[any]bool{}
		for _, o := range in.Options {
			valid[Get(o, "id")] = true
		}
		for _, id := range in.OptionIDs {
			if !valid[id] {
				return composer.StepResponse[any]{}, invalid("Shipping Options are invalid for cart.")
			}
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// ValidateCartShippingOptionsPriceStep mirrors validateCartShippingOptionsPriceStep.
var ValidateCartShippingOptionsPriceStep = composer.CreateStep("validate-cart-shipping-options-price",
	func(ctx composer.StepExecutionContext, options []*dal.Record) (composer.StepResponse[any], error) {
		var missing []string
		for _, o := range options {
			if Truthy(Get(o, "id")) {
				cp := Rec(o, "calculated_price")
				ok := false
				if cp != nil {
					var v any
					v, ok = cp.Get("calculated_amount")
					_ = v
				}
				if !ok {
					missing = append(missing, Str(o, "id"))
				}
			}
		}
		if len(missing) > 0 {
			ids := ""
			for i, m := range missing {
				if i > 0 {
					ids += ", "
				}
				ids += m
			}
			return composer.StepResponse[any]{}, invalid("Shipping options with IDs %s do not have a price", ids)
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// RemoveShippingMethodFromCartStep mirrors removeShippingMethodFromCartStep.
var RemoveShippingMethodFromCartStep = composer.CreateStep("remove-shipping-method",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		if len(ids) == 0 {
			return composer.NewStepResponse[any](nil, any([]any{})), nil
		}
		if _, err := CartOf(ctx.Container).Internal("ShippingMethod").SoftDelete(ctx.Context, ids); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := CartOf(ctx.Container).Internal("ShippingMethod").Restore(ctx.Context, ids)
		return err
	})

// AddShippingMethodToCartStep mirrors addShippingMethodToCartStep.
var AddShippingMethodToCartStep = composer.CreateStep("add-shipping-method-to-cart-step",
	func(ctx composer.StepExecutionContext, methods []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(methods) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]any{})), nil
		}
		created, err := CartOf(ctx.Container).Internal("ShippingMethod").Create(ctx.Context, methods)
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
		_, err := CartOf(ctx.Container).Internal("ShippingMethod").Delete(ctx.Context, ids)
		return err
	})

// UpdateShippingMethodsStep mirrors updateShippingMethodsStep.
var UpdateShippingMethodsStep = composer.CreateStep("update-shipping-methods-step",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[any], error) {
		if len(data) == 0 {
			return composer.NewStepResponse[any](nil, any([]*dal.Record{})), nil
		}
		svc := CartOf(ctx.Container)
		var ids, updates []any
		keys := []string{"id"}
		seen := map[string]bool{"id": true}
		for _, d := range data {
			ids = append(ids, Get(d, "id"))
			updates = append(updates, d)
			for _, k := range d.Keys() {
				if !seen[k] {
					seen[k] = true
					keys = append(keys, k)
				}
			}
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		before, err := svc.ListBySuffix(ctx.Context, "ShippingMethods", f, utilsmodulessdk.FindConfig{Select: keys})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		// updateShippingMethods has no @EmitEvents upstream
		if _, err := svc.Internal("ShippingMethod").Update(dal.WithoutEmit(ctx.Context), updates); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(before)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		before, _ := data.([]*dal.Record)
		if len(before) == 0 {
			return nil
		}
		var updates []any
		for _, b := range before {
			updates = append(updates, b)
		}
		_, err := CartOf(ctx.Container).Internal("ShippingMethod").Update(dal.WithoutEmit(ctx.Context), updates)
		return err
	})
