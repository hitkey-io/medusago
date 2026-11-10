package steps

import (
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func invalidData(msg string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, msg)
}

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	}
	return true
}

// ThrowIfNotDraftOrder mirrors throwIfNotDraftOrder.
func ThrowIfNotDraftOrder(order *dal.Record) error {
	draft, _ := get(order, "is_draft_order").(bool)
	if get(order, "status") != "draft" && !draft {
		return invalidData("Order is not a draft")
	}
	return nil
}

// ValidateDraftOrderChangeInput mirrors ValidateDraftOrderChangeStepInput.
type ValidateDraftOrderChangeInput struct {
	Order       *dal.Record
	OrderChange *dal.Record
}

// ValidateDraftOrderChangeStep mirrors validateDraftOrderChangeStep.
var ValidateDraftOrderChangeStep = composer.CreateStep("validate-draft-order-change",
	func(ctx composer.StepExecutionContext, in ValidateDraftOrderChangeInput) (composer.StepResponse[any], error) {
		if err := ThrowIfNotDraftOrder(in.Order); err != nil {
			return composer.StepResponse[any]{}, err
		}
		oc := in.OrderChange
		if oc == nil || oc.Len() == 0 {
			return composer.StepResponse[any]{}, invalidData("An active Order Change is required to proceed")
		}
		if truthy(get(oc, "canceled_at")) || truthy(get(oc, "confirmed_at")) || truthy(get(oc, "declined_at")) {
			return composer.StepResponse[any]{}, invalidData(fmt.Sprintf("Order change %v is not active to be modified", get(oc, "id")))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// ActionValidationInput mirrors the action validation step inputs.
type ActionValidationInput struct {
	OrderID     any
	ActionID    any
	OrderChange *dal.Record
}

func findAction(orderChange *dal.Record, id any) *dal.Record {
	for _, a := range recs(get(orderChange, "actions")) {
		if get(a, "id") == id {
			return a
		}
	}
	return nil
}

// FindOrderChangeAction returns the action of the order change with the id.
func FindOrderChangeAction(orderChange *dal.Record, id any) *dal.Record {
	return findAction(orderChange, id)
}

// ValidateDraftOrderUpdateActionItemStep mirrors validateDraftOrderUpdateActionItemStep.
var ValidateDraftOrderUpdateActionItemStep = composer.CreateStep("validate-draft-order-update-action-item",
	func(ctx composer.StepExecutionContext, in ActionValidationInput) (composer.StepResponse[any], error) {
		a := findAction(in.OrderChange, in.ActionID)
		if a == nil {
			return composer.StepResponse[any]{}, invalidData(fmt.Sprintf("No request to add item for order %v in order change %v", in.OrderID, get(in.OrderChange, "id")))
		}
		if get(a, "action") != "ITEM_ADD" {
			return composer.StepResponse[any]{}, invalidData(fmt.Sprintf("Action %v is not adding an item", get(a, "id")))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// ValidateDraftOrderRemoveActionItemStep mirrors validateDraftOrderRemoveActionItemStep.
var ValidateDraftOrderRemoveActionItemStep = composer.CreateStep("validate-draft-order-remove-action-item",
	func(ctx composer.StepExecutionContext, in ActionValidationInput) (composer.StepResponse[any], error) {
		a := findAction(in.OrderChange, in.ActionID)
		if a == nil {
			return composer.StepResponse[any]{}, invalidData(fmt.Sprintf("No item found for order %v in order change %v", in.OrderID, get(in.OrderChange, "id")))
		}
		if act := get(a, "action"); act != "ITEM_ADD" && act != "ITEM_UPDATE" {
			return composer.StepResponse[any]{}, invalidData(fmt.Sprintf("Action %v is not adding or updating an item", get(a, "id")))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// ValidateDraftOrderShippingMethodActionStep mirrors validateDraftOrderShippingMethodActionStep.
var ValidateDraftOrderShippingMethodActionStep = composer.CreateStep("validate-draft-order-shipping-method-action",
	func(ctx composer.StepExecutionContext, in ActionValidationInput) (composer.StepResponse[any], error) {
		a := findAction(in.OrderChange, in.ActionID)
		if a == nil {
			return composer.StepResponse[any]{}, invalidData(fmt.Sprintf("No shipping method found for order %v in order change %v", in.OrderID, get(in.OrderChange, "id")))
		}
		if get(a, "action") != "SHIPPING_ADD" {
			return composer.StepResponse[any]{}, invalidData(fmt.Sprintf("Action %v is not adding a shipping method", get(a, "id")))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// PromoCodesValidationInput mirrors ValidatePromoCodesToAddStepInput.
type PromoCodesValidationInput struct {
	PromoCodes []string
	Promotions []*dal.Record
}

func byCount(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func throwIfCodesAreMissing(codes []string, promotions []*dal.Record) error {
	var missing []string
	for _, c := range codes {
		found := false
		for _, p := range promotions {
			if get(p, "code") == c {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		return invalidData(byCount(len(missing), fmt.Sprintf("Promotion code %q not found", missing[0]),
			fmt.Sprintf("Promotion codes \"%s\" not found", strings.Join(missing, "\", \""))))
	}
	return nil
}

func throwIfCodesAreInactive(codes []string, promotions []*dal.Record) error {
	var inactive []string
	for _, c := range codes {
		for _, p := range promotions {
			if get(p, "code") == c && get(p, "status") != "active" {
				inactive = append(inactive, c)
				break
			}
		}
	}
	if len(inactive) > 0 {
		return invalidData(byCount(len(inactive), fmt.Sprintf("Promotion code %q is not active", inactive[0]),
			fmt.Sprintf("Promotion codes \"%s\" are not active", strings.Join(inactive, "\", \""))))
	}
	return nil
}

// ValidatePromoCodesToAddStep mirrors validatePromoCodesToAddStep.
var ValidatePromoCodesToAddStep = composer.CreateStep("validate-promo-codes-to-add",
	func(ctx composer.StepExecutionContext, in PromoCodesValidationInput) (composer.StepResponse[any], error) {
		if err := throwIfCodesAreMissing(in.PromoCodes, in.Promotions); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := throwIfCodesAreInactive(in.PromoCodes, in.Promotions); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// ValidatePromoCodesToRemoveStep mirrors validatePromoCodesToRemoveStep.
var ValidatePromoCodesToRemoveStep = composer.CreateStep("validate-promo-codes-to-remove",
	func(ctx composer.StepExecutionContext, in PromoCodesValidationInput) (composer.StepResponse[any], error) {
		if err := throwIfCodesAreMissing(in.PromoCodes, in.Promotions); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

func shippingMethods(c composer.Container) *utilsmodulessdk.InternalService {
	return orderOf(c).Internal("OrderShippingMethod")
}

func toAnyList(list []*dal.Record) []any {
	out := make([]any, len(list))
	for i, r := range list {
		out[i] = r
	}
	return out
}

// ShippingMethodRestore is one shipping method of restoreDraftOrderShippingMethodsStep.
type ShippingMethodRestore struct {
	ID     any
	Before *dal.Record
	After  *dal.Record
}

func methodState(id any, state *dal.Record) *dal.Record {
	r := dal.NewRecord()
	r.Set("id", id)
	if v, ok := state.Get("shipping_option_id"); ok {
		r.Set("shipping_option_id", v)
	}
	if v, ok := state.Get("amount"); ok {
		r.Set("amount", v)
	}
	return r
}

// RestoreDraftOrderShippingMethodsStep mirrors restoreDraftOrderShippingMethodsStep.
var RestoreDraftOrderShippingMethodsStep = composer.CreateStep("restore-draft-order-shipping-methods",
	func(ctx composer.StepExecutionContext, in []ShippingMethodRestore) (composer.StepResponse[any], error) {
		data := make([]*dal.Record, len(in))
		for i, m := range in {
			data[i] = methodState(m.ID, m.Before)
		}
		if _, err := shippingMethods(ctx.Container).Update(ctx.Context, toAnyList(data)); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(in)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		in, _ := data.([]ShippingMethodRestore)
		if in == nil {
			return nil
		}
		list := make([]*dal.Record, len(in))
		for i, m := range in {
			list[i] = methodState(m.ID, m.After)
		}
		_, err := shippingMethods(ctx.Container).Update(ctx.Context, toAnyList(list))
		return err
	})

// UpdateDraftOrderShippingMethodInput mirrors UpdateDraftOrderShippingMethodStepInput.
type UpdateDraftOrderShippingMethodInput struct {
	OrderID          any
	ShippingMethodID any
	ShippingOptionID any
	Amount           any
}

// UpdateDraftOrderShippingMethodResult is { before, after }.
type UpdateDraftOrderShippingMethodResult struct {
	Before *dal.Record
	After  *dal.Record
}

// UpdateDraftOrderShippingMethodStep mirrors updateDraftOrderShippingMethodStep.
var UpdateDraftOrderShippingMethodStep = composer.CreateStep("update-draft-order-shipping-method",
	func(ctx composer.StepExecutionContext, in UpdateDraftOrderShippingMethodInput) (composer.StepResponse[UpdateDraftOrderShippingMethodResult], error) {
		f := dal.NewRecord()
		f.Set("id", in.ShippingMethodID)
		take := 1
		list, err := shippingMethods(ctx.Container).List(ctx.Context, f, utilsmodulessdk.FindConfig{Take: &take, Select: []string{"id", "shipping_option_id", "amount", "is_custom_amount"}})
		if err != nil {
			return composer.StepResponse[UpdateDraftOrderShippingMethodResult]{}, err
		}
		if len(list) == 0 {
			return composer.StepResponse[UpdateDraftOrderShippingMethodResult]{}, invalidData(fmt.Sprintf("A shipping method with id %v was not found", in.ShippingMethodID))
		}
		before := list[0]
		u := dal.NewRecord()
		u.Set("id", in.ShippingMethodID)
		// shipping_option_id: undefined is assigned as is upstream, so an
		// omitted option clears the column
		u.Set("shipping_option_id", in.ShippingOptionID)
		if in.Amount != nil {
			u.Set("amount", in.Amount)
			u.Set("is_custom_amount", true)
		}
		res, err := shippingMethods(ctx.Container).Update(ctx.Context, []any{u})
		if err != nil {
			return composer.StepResponse[UpdateDraftOrderShippingMethodResult]{}, err
		}
		updated, _ := res.([]*dal.Record)
		var after *dal.Record
		if len(updated) > 0 {
			after = updated[0]
		}
		return composer.NewStepResponse(UpdateDraftOrderShippingMethodResult{Before: before, After: after}, any(before)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		before, _ := data.(*dal.Record)
		if before == nil {
			return nil
		}
		_, err := shippingMethods(ctx.Container).Update(ctx.Context, []any{before})
		return err
	})
