package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// AddShippingMethodInput mirrors AddShippingMethodToCartWorkflowInput.
type AddShippingMethodInput struct {
	CartID         string
	Options        []*dal.Record // { id, data }
	AdditionalData *dal.Record
}

func findOption(options []*dal.Record, id any) *dal.Record {
	for _, o := range options {
		if Get(o, "id") == id {
			return o
		}
	}
	return nil
}

// AddShippingMethodToCartWorkflow mirrors addShippingMethodToCartWorkflow.
var AddShippingMethodToCartWorkflow = composer.CreateWorkflow("add-shipping-method-to-cart",
	func(e *composer.Execution, in AddShippingMethodInput) (any, error) {
		cart, err := fetchCart(e, "use-remote-query", in.CartID, CartFieldsForRefreshSteps, true)
		if err != nil {
			return nil, err
		}
		if _, err := steps.ValidateCartStep.Run(e, cart); err != nil {
			return nil, err
		}
		var optionIDs []any
		for _, o := range in.Options {
			optionIDs = append(optionIDs, Get(o, "id"))
		}
		options, err := ListShippingOptionsForCartWithPricingWorkflow.RunAsStep(e, PricedOptionsInput{CartID: Str(cart, "id"), Options: in.Options, AdditionalData: in.AdditionalData})
		if err != nil {
			return nil, err
		}
		if _, err := steps.ValidateCartShippingOptionsStep.Run(e, struct {
			OptionIDs []any
			Options   []*dal.Record
		}{optionIDs, options}); err != nil {
			return nil, err
		}
		if _, err := steps.ValidateCartShippingOptionsPriceStep.Run(e, options); err != nil {
			return nil, err
		}
		var validateInput []*dal.Record
		for _, o := range in.Options {
			so := findOption(options, Get(o, "id"))
			optionData := Rec(so, "data")
			if optionData == nil {
				optionData = dal.NewRecord()
			}
			methodData := Rec(o, "data")
			if methodData == nil {
				methodData = dal.NewRecord()
			}
			c := Copy(cart)
			from := Rec(so, "stock_location")
			if from == nil {
				from = dal.NewRecord()
			}
			c.Set("from_location", from)
			v := dal.NewRecord()
			v.Set("id", Get(o, "id"))
			v.Set("provider_id", Get(so, "provider_id"))
			v.Set("option_data", optionData)
			v.Set("method_data", methodData)
			v.Set("context", c)
			validateInput = append(validateInput, v)
		}
		validated, err := steps.ValidateAndReturnShippingMethodsDataStep.Run(e, validateInput)
		if err != nil {
			return nil, err
		}
		var methods []*dal.Record
		for _, o := range in.Options {
			so := findOption(options, Get(o, "id"))
			cp := Rec(so, "calculated_price")
			if cp == nil {
				return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Shipping option with ID "+Str(so, "id")+" do not have a price")
			}
			var data any
			for _, v := range validated {
				if d, ok := v.Get(Str(o, "id")); ok && Truthy(d) {
					data = d
					break
				}
			}
			if data == nil {
				data = dal.NewRecord()
			}
			m := dal.NewRecord()
			m.Set("shipping_option_id", Get(so, "id"))
			m.Set("amount", Get(cp, "calculated_amount"))
			m.Set("is_tax_inclusive", Truthy(Get(cp, "is_calculated_price_tax_inclusive")))
			m.Set("data", data)
			m.Set("name", Get(so, "name"))
			m.Set("cart_id", in.CartID)
			methods = append(methods, m)
		}
		incoming := map[any]bool{}
		for _, o := range in.Options {
			if p := Get(findOption(options, Get(o, "id")), "shipping_profile_id"); Truthy(p) {
				incoming[p] = true
			}
		}
		var colliding []any
		for _, sm := range Records(Get(cart, "shipping_methods")) {
			p := Get(Rec(sm, "shipping_option"), "shipping_profile_id")
			if !Truthy(p) || incoming[p] {
				colliding = append(colliding, Get(sm, "id"))
			}
		}
		var created []*dal.Record
		err = composer.Parallelize(
			func() error { _, err := steps.RemoveShippingMethodFromCartStep.Run(e, colliding); return err },
			func() error {
				var err error
				created, err = steps.AddShippingMethodToCartStep.Run(e, methods)
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		if _, err := RefreshCartItemsWorkflow.RunAsStep(e, RefreshCartItemsInput{CartID: Str(cart, "id"), ShippingMethods: created, AdditionalData: in.AdditionalData}); err != nil {
			return nil, err
		}
		ev := dal.NewRecord()
		ev.Set("id", in.CartID)
		_, err = commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "cart.updated", Data: ev})
		return nil, err
	})
