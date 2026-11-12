package workflows

import (
	"fmt"
	"time"

	cartsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	draftordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/draft-order/steps"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	orderworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// DraftOrderFieldsForRefreshSteps mirrors draftOrderFieldsForRefreshSteps.
var DraftOrderFieldsForRefreshSteps = []string{"id", "is_draft_order", "status", "currency_code", "email", "metadata", "sales_channel_id",
	"region_id", "locale", "region.*", "items.*", "items.product.id", "items.product.collection_id", "items.product.categories.id",
	"items.product.tags.id", "items.variant.id", "items.variant.product.id", "items.variant.weight", "items.variant.length",
	"items.variant.height", "items.variant.width", "items.variant.material", "items.adjustments.*", "items.tax_lines.*",
	"shipping_address.*", "shipping_methods.*", "shipping_methods.adjustments.*", "shipping_methods.tax_lines.*", "customer.*",
	"customer.groups.*", "promotions.id", "promotions.code", "subtotal", "item_total", "total", "item_subtotal"}

// draftActiveChange mirrors the "order-change-query" remote query of the
// draft order edit workflows (pending or requested change, list false).
func draftActiveChange(e *composer.Execution, orderID any, fields []string) (*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("order_id", orderID)
	f.Set("status", []any{"pending", "requested"})
	v := dal.NewRecord()
	v.Set("filters", f)
	list := false
	res, err := commonsteps.UseRemoteQueryStep.Config("order-change-query", composer.StepOptions{}).Run(e, commonsteps.RemoteStepInput{
		EntryPoint: "order_change", Fields: fields, Variables: v, List: &list,
	})
	if err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, nil
	}
	return rec(res.Data[0]), nil
}

func validateDraftChange(e *composer.Execution, order, orderChange *dal.Record) error {
	_, err := draftordersteps.ValidateDraftOrderChangeStep.Run(e, draftordersteps.ValidateDraftOrderChangeInput{Order: order, OrderChange: orderChange})
	return err
}

func preview(e *composer.Execution, orderID any) (*dal.Record, error) {
	return ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(orderID))
}

func promotionCodes(order *dal.Record) []string {
	out := []string{}
	for _, p := range recs(g(order, "promotions")) {
		if c, ok := g(p, "code").(string); ok {
			out = append(out, c)
		}
	}
	return out
}

func computeIfPromotions(e *composer.Execution, order *dal.Record, orderID any) error {
	if len(promotionCodes(order)) == 0 {
		return nil
	}
	_, err := ComputeDraftOrderAdjustmentsWorkflow.RunAsStep(e, orderID)
	return err
}

func setIfPresent(dst, src *dal.Record, keys ...string) {
	for _, k := range keys {
		if v, ok := src.Get(k); ok {
			dst.Set(k, v)
		}
	}
}

func coalesce(a, b any) any {
	if a != nil {
		return a
	}
	return b
}

// ComputeDraftOrderAdjustmentsWorkflow mirrors computeDraftOrderAdjustmentsWorkflow.
var ComputeDraftOrderAdjustmentsWorkflow = composer.CreateWorkflow("compute-draft-order-adjustments",
	func(e *composer.Execution, orderID any) (any, error) {
		if err := acquireLock(e, orderID); err != nil {
			return nil, err
		}
		order, err := remoteOne(e, "order-query", "orders", orderID, DraftOrderFieldsForRefreshSteps)
		if err != nil {
			return nil, err
		}
		orderChange, err := draftActiveChange(e, orderID, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if err := validateDraftChange(e, order, orderChange); err != nil {
			return nil, err
		}
		hookIn := dal.NewRecord()
		hookIn.Set("order", order)
		hookIn.Set("orderChange", orderChange)
		if _, err := composer.CreateHookWithResult(e, "setPromotionContext", hookIn); err != nil {
			return nil, err
		}
		toDelete := []any{}
		for _, a := range recs(g(orderChange, "actions")) {
			if g(a, "action") == "ITEM_ADJUSTMENTS_REPLACE" {
				toDelete = append(toDelete, g(a, "id"))
			}
		}
		if len(toDelete) > 0 {
			if _, err := ordersteps.DeleteOrderChangeActionsStep.Run(e, toDelete); err != nil {
				return nil, err
			}
		}
		previewed, err := preview(e, orderID)
		if err != nil {
			return nil, err
		}
		replaceAction := func(item *dal.Record, adjustments []any) *dal.Record {
			a := dal.NewRecord()
			a.Set("order_id", g(order, "id"))
			a.Set("order_change_id", g(orderChange, "id"))
			a.Set("version", g(orderChange, "version"))
			a.Set("action", "ITEM_ADJUSTMENTS_REPLACE")
			d := dal.NewRecord()
			d.Set("reference_id", g(item, "id"))
			d.Set("adjustments", adjustments)
			a.Set("details", d)
			return a
		}
		promotions, isList := g(order, "promotions").([]any)
		if isList && len(promotions) == 0 {
			actions := []*dal.Record{}
			for _, item := range recs(g(previewed, "items")) {
				actions = append(actions, replaceAction(item, []any{}))
			}
			if _, err := orderworkflows.CreateOrderChangeActionsWorkflow.RunAsStep(e, actions); err != nil {
				return nil, err
			}
		}
		if codes := promotionCodes(order); len(recs(g(order, "promotions"))) > 0 {
			computeCtx, err := orderworkflows.PrepareOrderComputeActionContextStep.Run(e, orderworkflows.PrepareOrderComputeActionContextInput{Order: order, PreviewedOrder: previewed})
			if err != nil {
				return nil, err
			}
			actions, err := cartsteps.GetActionsToComputeFromPromotionsStep.Run(e, cartsteps.ActionsInput{ComputeActionContext: computeCtx, PromotionCodesToApply: codes})
			if err != nil {
				return nil, err
			}
			prepared, err := cartsteps.PrepareAdjustmentsFromPromotionActionsStep.Run(e, actions)
			if err != nil {
				return nil, err
			}
			changeActions := []*dal.Record{}
			for _, item := range recs(g(previewed, "items")) {
				adjustments := []any{}
				for _, adj := range prepared.LineItemAdjustmentsToCreate {
					if g(adj, "item_id") == g(item, "id") {
						adjustments = append(adjustments, adj)
					}
				}
				changeActions = append(changeActions, replaceAction(item, adjustments))
			}
			if _, err := orderworkflows.CreateOrderChangeActionsWorkflow.RunAsStep(e, changeActions); err != nil {
				return nil, err
			}
		}
		return nil, releaseLock(e, orderID)
	})

// BeginDraftOrderEditWorkflow mirrors beginDraftOrderEditWorkflow.
var BeginDraftOrderEditWorkflow = composer.CreateWorkflow("begin-draft-order-edit",
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		if err := acquireLock(e, g(in, "order_id")); err != nil {
			return nil, err
		}
		orderID := g(in, "order_id")
		order, err := remoteOne(e, "order-query", "orders", orderID, []string{"id", "status", "is_draft_order"})
		if err != nil {
			return nil, err
		}
		if _, err := draftordersteps.ValidateDraftOrderStep.Run(e, order); err != nil {
			return nil, err
		}
		c := dal.NewRecord()
		c.Set("change_type", "edit")
		c.Set("order_id", orderID)
		setIfPresent(c, in, "created_by", "description", "internal_note")
		if _, err := ordersteps.CreateOrderChangeStep.Run(e, c); err != nil {
			return nil, err
		}
		if err := releaseLock(e, g(in, "order_id")); err != nil {
			return nil, err
		}
		return preview(e, orderID)
	})

// DraftOrderItemsInput mirrors OrderEditAddNewItemWorkflowInput and
// OrderEditUpdateItemQuantityWorkflowInput.
type DraftOrderItemsInput struct {
	OrderID any
	Items   []*dal.Record
}

// AddDraftOrderItemsWorkflow mirrors addDraftOrderItemsWorkflow.
var AddDraftOrderItemsWorkflow = composer.CreateWorkflow("add-draft-order-items",
	func(e *composer.Execution, in DraftOrderItemsInput) (*dal.Record, error) {
		if err := acquireLock(e, in.OrderID); err != nil {
			return nil, err
		}
		order, err := remoteOne(e, "order-query", "orders", in.OrderID, DraftOrderFieldsForRefreshSteps)
		if err != nil {
			return nil, err
		}
		orderChange, err := draftActiveChange(e, in.OrderID, []string{"id", "status", "version"})
		if err != nil {
			return nil, err
		}
		if err := validateDraftChange(e, order, orderChange); err != nil {
			return nil, err
		}
		lineItems, err := orderworkflows.AddOrderLineItemsWorkflow.RunAsStep(e, orderworkflows.AddOrderLineItemsInput{OrderID: g(order, "id"), Items: in.Items})
		if err != nil {
			return nil, err
		}
		itemIDs := []any{}
		for _, li := range lineItems {
			itemIDs = append(itemIDs, g(li, "id"))
		}
		if _, err := orderworkflows.UpdateOrderTaxLinesWorkflow.RunAsStep(e, orderworkflows.UpdateOrderTaxLinesInput{OrderID: g(order, "id"), ItemIDs: itemIDs}); err != nil {
			return nil, err
		}
		actions := make([]*dal.Record, 0, len(in.Items))
		for i, item := range in.Items {
			a := dal.NewRecord()
			a.Set("order_change_id", g(orderChange, "id"))
			a.Set("order_id", g(order, "id"))
			a.Set("version", g(orderChange, "version"))
			a.Set("action", "ITEM_ADD")
			setIfPresent(a, item, "internal_note")
			d := dal.NewRecord()
			d.Set("reference_id", g(lineItems[i], "id"))
			d.Set("quantity", g(item, "quantity"))
			d.Set("unit_price", coalesce(g(item, "unit_price"), g(lineItems[i], "unit_price")))
			d.Set("compare_at_unit_price", coalesce(g(item, "compare_at_unit_price"), g(lineItems[i], "compare_at_unit_price")))
			setIfPresent(d, item, "allow_backorder", "metadata")
			a.Set("details", d)
			actions = append(actions, a)
		}
		if _, err := orderworkflows.CreateOrderChangeActionsWorkflow.RunAsStep(e, actions); err != nil {
			return nil, err
		}
		if _, err := RefreshPendingDraftOrderShippingMethodsWorkflow.RunAsStep(e, in.OrderID); err != nil {
			return nil, err
		}
		if err := computeIfPromotions(e, order, in.OrderID); err != nil {
			return nil, err
		}
		if err := releaseLock(e, in.OrderID); err != nil {
			return nil, err
		}
		return preview(e, in.OrderID)
	})

// DraftOrderActionInput mirrors the { order_id, action_id, data } inputs.
type DraftOrderActionInput struct {
	OrderID  any
	ActionID any
	Data     *dal.Record
}

// UpdateDraftOrderActionItemWorkflow mirrors updateDraftOrderActionItemWorkflow.
var UpdateDraftOrderActionItemWorkflow = composer.CreateWorkflow("update-draft-order-action-item",
	func(e *composer.Execution, in DraftOrderActionInput) (*dal.Record, error) {
		if err := acquireLock(e, in.OrderID); err != nil {
			return nil, err
		}
		order, err := remoteOne(e, "order-query", "orders", in.OrderID, DraftOrderFieldsForRefreshSteps)
		if err != nil {
			return nil, err
		}
		orderChange, err := draftActiveChange(e, in.OrderID, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if err := validateDraftChange(e, order, orderChange); err != nil {
			return nil, err
		}
		if _, err := draftordersteps.ValidateDraftOrderUpdateActionItemStep.Run(e, draftordersteps.ActionValidationInput{OrderID: in.OrderID, ActionID: in.ActionID, OrderChange: orderChange}); err != nil {
			return nil, err
		}
		original := rec(g(draftordersteps.FindOrderChangeAction(orderChange, in.ActionID), "details"))
		u := dal.NewRecord()
		u.Set("id", in.ActionID)
		d := dal.NewRecord()
		for _, k := range []string{"quantity", "unit_price", "compare_at_unit_price"} {
			if v := coalesce(g(in.Data, k), g(original, k)); v != nil {
				d.Set(k, v)
			}
		}
		u.Set("details", d)
		setIfPresent(u, in.Data, "internal_note")
		if _, err := ordersteps.UpdateOrderChangeActionsStep.Run(e, []*dal.Record{u}); err != nil {
			return nil, err
		}
		if _, err := RefreshPendingDraftOrderShippingMethodsWorkflow.RunAsStep(e, in.OrderID); err != nil {
			return nil, err
		}
		if err := computeIfPromotions(e, order, in.OrderID); err != nil {
			return nil, err
		}
		if err := releaseLock(e, in.OrderID); err != nil {
			return nil, err
		}
		return preview(e, in.OrderID)
	})

// RemoveDraftOrderActionItemWorkflow mirrors removeDraftOrderActionItemWorkflow.
var RemoveDraftOrderActionItemWorkflow = composer.CreateWorkflow("remove-draft-order-action-item",
	func(e *composer.Execution, in DraftOrderActionInput) (*dal.Record, error) {
		if err := acquireLock(e, in.OrderID); err != nil {
			return nil, err
		}
		order, err := remoteOne(e, "order-query", "orders", in.OrderID, []string{"id", "status", "is_draft_order", "canceled_at", "items.*"})
		if err != nil {
			return nil, err
		}
		orderChange, err := draftActiveChange(e, in.OrderID, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if err := validateDraftChange(e, order, orderChange); err != nil {
			return nil, err
		}
		if _, err := draftordersteps.ValidateDraftOrderRemoveActionItemStep.Run(e, draftordersteps.ActionValidationInput{OrderID: in.OrderID, ActionID: in.ActionID, OrderChange: orderChange}); err != nil {
			return nil, err
		}
		if _, err := ordersteps.DeleteOrderChangeActionsStep.Run(e, []any{in.ActionID}); err != nil {
			return nil, err
		}
		refetched, err := remoteOne(e, "refetched-order-query", "orders", in.OrderID, DraftOrderFieldsForRefreshSteps)
		if err != nil {
			return nil, err
		}
		if _, err := RefreshPendingDraftOrderShippingMethodsWorkflow.RunAsStep(e, in.OrderID); err != nil {
			return nil, err
		}
		if err := computeIfPromotions(e, refetched, in.OrderID); err != nil {
			return nil, err
		}
		if err := releaseLock(e, in.OrderID); err != nil {
			return nil, err
		}
		return preview(e, in.OrderID)
	})

// UpdateDraftOrderItemWorkflow mirrors updateDraftOrderItemWorkflow.
var UpdateDraftOrderItemWorkflow = composer.CreateWorkflow("update-draft-order-item",
	func(e *composer.Execution, in DraftOrderItemsInput) (*dal.Record, error) {
		if err := acquireLock(e, in.OrderID); err != nil {
			return nil, err
		}
		order, err := remoteOne(e, "order-query", "orders", in.OrderID, DraftOrderFieldsForRefreshSteps)
		if err != nil {
			return nil, err
		}
		orderChange, err := draftActiveChange(e, in.OrderID, []string{"id", "status", "version"})
		if err != nil {
			return nil, err
		}
		if err := validateDraftChange(e, order, orderChange); err != nil {
			return nil, err
		}
		actions := make([]*dal.Record, 0, len(in.Items))
		for _, item := range in.Items {
			var existing *dal.Record
			for _, oi := range recs(g(order, "items")) {
				if g(oi, "id") == g(item, "id") {
					existing = oi
					break
				}
			}
			a := dal.NewRecord()
			a.Set("order_change_id", g(orderChange, "id"))
			a.Set("order_id", g(order, "id"))
			a.Set("version", g(orderChange, "version"))
			a.Set("action", "ITEM_UPDATE")
			setIfPresent(a, item, "internal_note")
			d := dal.NewRecord()
			d.Set("reference_id", g(item, "id"))
			d.Set("quantity", g(item, "quantity"))
			setIfPresent(d, item, "unit_price", "compare_at_unit_price")
			d.Set("quantity_diff", totals.JSONNumber(totals.Sub(g(item, "quantity"), g(existing, "quantity"))))
			setIfPresent(d, item, "metadata")
			a.Set("details", d)
			actions = append(actions, a)
		}
		if _, err := orderworkflows.CreateOrderChangeActionsWorkflow.RunAsStep(e, actions); err != nil {
			return nil, err
		}
		if _, err := RefreshPendingDraftOrderShippingMethodsWorkflow.RunAsStep(e, in.OrderID); err != nil {
			return nil, err
		}
		if err := computeIfPromotions(e, order, in.OrderID); err != nil {
			return nil, err
		}
		if err := releaseLock(e, in.OrderID); err != nil {
			return nil, err
		}
		return preview(e, in.OrderID)
	})

// DraftOrderPromotionsInput mirrors Add/RemoveDraftOrderPromotionsWorkflowInput.
type DraftOrderPromotionsInput struct {
	OrderID    any
	PromoCodes []string
}

func draftPromotionsWorkflow(name, action, changeAction, detailKey string, validate *composer.Step[draftordersteps.PromoCodesValidationInput, any]) *composer.Workflow[DraftOrderPromotionsInput, *dal.Record] {
	return composer.CreateWorkflow(name,
		func(e *composer.Execution, in DraftOrderPromotionsInput) (*dal.Record, error) {
			if err := acquireLock(e, in.OrderID); err != nil {
				return nil, err
			}
			order, err := remoteOne(e, "order-query", "orders", in.OrderID, DraftOrderFieldsForRefreshSteps)
			if err != nil {
				return nil, err
			}
			orderChange, err := draftActiveChange(e, in.OrderID, []string{"id", "status", "version"})
			if err != nil {
				return nil, err
			}
			if err := validateDraftChange(e, order, orderChange); err != nil {
				return nil, err
			}
			codes := make([]any, len(in.PromoCodes))
			for i, c := range in.PromoCodes {
				codes[i] = c
			}
			f := dal.NewRecord()
			f.Set("code", codes)
			v := dal.NewRecord()
			v.Set("filters", f)
			res, err := commonsteps.UseRemoteQueryStep.Config("promotions-query", composer.StepOptions{}).Run(e, commonsteps.RemoteStepInput{
				EntryPoint: "promotion", Fields: []string{"id", "code", "status"}, Variables: v,
			})
			if err != nil {
				return nil, err
			}
			promotions := recs(res.Data)
			if _, err := validate.Run(e, draftordersteps.PromoCodesValidationInput{PromoCodes: in.PromoCodes, Promotions: promotions}); err != nil {
				return nil, err
			}
			if _, err := draftordersteps.UpdateDraftOrderPromotionsStep.Run(e, draftordersteps.UpdateDraftOrderPromotionsInput{ID: in.OrderID, PromoCodes: in.PromoCodes, Action: action}); err != nil {
				return nil, err
			}
			actions := []*dal.Record{}
			for _, p := range promotions {
				a := dal.NewRecord()
				a.Set("action", changeAction)
				a.Set("reference", "order_promotion")
				a.Set("order_change_id", g(orderChange, "id"))
				a.Set("reference_id", g(p, "id"))
				a.Set("order_id", g(order, "id"))
				d := dal.NewRecord()
				d.Set(detailKey, g(p, "code"))
				a.Set("details", d)
				actions = append(actions, a)
			}
			if _, err := orderworkflows.CreateOrderChangeActionsWorkflow.RunAsStep(e, actions); err != nil {
				return nil, err
			}
			if _, err := ComputeDraftOrderAdjustmentsWorkflow.RunAsStep(e, in.OrderID); err != nil {
				return nil, err
			}
			if err := releaseLock(e, in.OrderID); err != nil {
				return nil, err
			}
			return preview(e, in.OrderID)
		})
}

// AddDraftOrderPromotionWorkflow mirrors addDraftOrderPromotionWorkflow.
var AddDraftOrderPromotionWorkflow = draftPromotionsWorkflow("add-draft-order-promotion", cartsteps.PromotionActionAdd, "PROMOTION_ADD", "added_code", draftordersteps.ValidatePromoCodesToAddStep)

// RemoveDraftOrderPromotionsWorkflow mirrors removeDraftOrderPromotionsWorkflow.
var RemoveDraftOrderPromotionsWorkflow = draftPromotionsWorkflow("remove-draft-order-promotions", cartsteps.PromotionActionRemove, "PROMOTION_REMOVE", "removed_code", draftordersteps.ValidatePromoCodesToRemoveStep)

// FetchShippingOptionForDraftOrderInput mirrors FetchShippingOptionForDraftOrderWorkflowInput.
type FetchShippingOptionForDraftOrderInput struct {
	OrderID          any
	ShippingOptionID any
	CurrencyCode     any
}

// FetchShippingOptionForDraftOrderWorkflow mirrors
// fetchShippingOptionForDraftOrderWorkflow.
var FetchShippingOptionForDraftOrderWorkflow = composer.CreateWorkflow("fetch-shipping-option-for-draft-order",
	func(e *composer.Execution, in FetchShippingOptionForDraftOrderInput) (*dal.Record, error) {
		f := dal.NewRecord()
		f.Set("id", in.ShippingOptionID)
		initial, err := graphOne(e, "shipping-option-query", "shipping_option", f, []string{"id", "price_type"}, false)
		if err != nil {
			return nil, err
		}
		isCalculated := g(initial, "price_type") == "calculated"
		hookIn := dal.NewRecord()
		hookIn.Set("input", in)
		if _, err := composer.CreateHookWithResult(e, "setCalculatedShippingPricingContext", hookIn); err != nil {
			return nil, err
		}
		calculated, err := composer.When(e, "option-calculated", isCalculated, func() (*dal.Record, error) {
			return fetchCalculatedDraftOrderOption(e, in)
		})
		if err != nil {
			return nil, err
		}
		pricingResult, err := composer.CreateHookWithResult(e, "setPricingContext", in)
		if err != nil {
			return nil, err
		}
		// { ...setPricingContextResult, currency_code }
		pc := dal.NewRecord()
		if r, ok := pricingResult.(*dal.Record); ok {
			for _, k := range r.Keys() {
				v, _ := r.Get(k)
				pc.Set(k, v)
			}
		}
		pc.Set("currency_code", in.CurrencyCode)
		flat, err := composer.When(e, "option-flat", !isCalculated, func() (*dal.Record, error) {
			qc := dal.NewRecord()
			qc.Set("calculated_price", query.QueryContext(pc))
			isList := false
			res, err := commonsteps.UseQueryGraphStep.Config("flat-rate-option", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
				GraphInput: query.GraphInput{Entity: "shipping_option", Fields: []string{"id", "name", "calculated_price.calculated_amount",
					"calculated_price.is_calculated_price_tax_inclusive"}, Filters: f, Context: qc},
				IsList: &isList,
			})
			if err != nil {
				return nil, err
			}
			return rec(res.Data), nil
		})
		if err != nil {
			return nil, err
		}
		if calculated != nil {
			return calculated, nil
		}
		return flat, nil
	})

// AddDraftOrderShippingMethodsWorkflow mirrors addDraftOrderShippingMethodsWorkflow.
var AddDraftOrderShippingMethodsWorkflow = composer.CreateWorkflow("add-draft-order-shipping-methods",
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		if err := acquireLock(e, g(in, "order_id")); err != nil {
			return nil, err
		}
		orderID := g(in, "order_id")
		order, err := remoteOne(e, "order-query", "orders", orderID, DraftOrderFieldsForRefreshSteps)
		if err != nil {
			return nil, err
		}
		orderChange, err := draftActiveChange(e, orderID, []string{"id", "status", "version"})
		if err != nil {
			return nil, err
		}
		if err := validateDraftChange(e, order, orderChange); err != nil {
			return nil, err
		}
		option, err := FetchShippingOptionForDraftOrderWorkflow.RunAsStep(e, FetchShippingOptionForDraftOrderInput{
			OrderID: g(order, "id"), ShippingOptionID: g(in, "shipping_option_id"), CurrencyCode: g(order, "currency_code"),
		})
		if err != nil {
			return nil, err
		}
		if option, err = orderworkflows.TranslateShippingOption(e, option, g(order, "locale")); err != nil {
			return nil, err
		}
		related := dal.NewRecord()
		related.Set("order_id", g(order, "id"))
		customPrice := g(in, "custom_amount")
		method := orderworkflows.PrepareShippingMethod("", option, orderChange, related, in, customPrice)
		created, err := ordersteps.CreateOrderShippingMethodsStep.Run(e, []*dal.Record{method})
		if err != nil {
			return nil, err
		}
		methodIDs := []any{}
		for _, m := range created {
			methodIDs = append(methodIDs, g(m, "id"))
		}
		if _, err := orderworkflows.UpdateOrderTaxLinesWorkflow.RunAsStep(e, orderworkflows.UpdateOrderTaxLinesInput{OrderID: g(order, "id"), ShippingMethodIDs: methodIDs}); err != nil {
			return nil, err
		}
		a := dal.NewRecord()
		a.Set("action", "SHIPPING_ADD")
		a.Set("reference", "order_shipping_method")
		a.Set("order_change_id", g(orderChange, "id"))
		a.Set("reference_id", g(created[0], "id"))
		a.Set("amount", coalesce(customPrice, g(rec(g(option, "calculated_price")), "calculated_amount")))
		a.Set("order_id", g(order, "id"))
		if _, err := orderworkflows.CreateOrderChangeActionsWorkflow.RunAsStep(e, []*dal.Record{a}); err != nil {
			return nil, err
		}
		if err := computeIfPromotions(e, order, orderID); err != nil {
			return nil, err
		}
		if err := releaseLock(e, g(in, "order_id")); err != nil {
			return nil, err
		}
		return preview(e, g(order, "id"))
	})

// UpdateDraftOrderActionShippingMethodWorkflow mirrors updateDraftOrderActionShippingMethodWorkflow.
var UpdateDraftOrderActionShippingMethodWorkflow = composer.CreateWorkflow("update-draft-order-action-shipping-method",
	func(e *composer.Execution, in DraftOrderActionInput) (*dal.Record, error) {
		if err := acquireLock(e, in.OrderID); err != nil {
			return nil, err
		}
		order, err := remoteOne(e, "order-query", "orders", in.OrderID, DraftOrderFieldsForRefreshSteps)
		if err != nil {
			return nil, err
		}
		orderChange, err := draftActiveChange(e, in.OrderID, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if err := validateDraftChange(e, order, orderChange); err != nil {
			return nil, err
		}
		v, hasCustom := in.Data.Get("custom_amount")
		hasOptions := hasCustom && v == nil
		shippingOptions, err := composer.When(e, "get-shipping-options", hasOptions, func() ([]*dal.Record, error) {
			original := draftordersteps.FindOrderChangeAction(orderChange, in.ActionID)
			method, err := remoteOne(e, "fetch-shipping-method", "order_shipping_method", g(original, "reference_id"), []string{"id", "shipping_option_id"})
			if err != nil {
				return nil, err
			}
			option, err := FetchShippingOptionForDraftOrderWorkflow.RunAsStep(e, FetchShippingOptionForDraftOrderInput{
				OrderID: in.OrderID, ShippingOptionID: g(method, "shipping_option_id"), CurrencyCode: g(order, "currency_code"),
			})
			if err != nil {
				return nil, err
			}
			return []*dal.Record{option}, nil
		})
		if err != nil {
			return nil, err
		}
		var option *dal.Record
		if len(shippingOptions) > 0 {
			option = shippingOptions[0]
		}
		if _, err := draftordersteps.ValidateDraftOrderShippingMethodActionStep.Run(e, draftordersteps.ActionValidationInput{OrderID: in.OrderID, ActionID: in.ActionID, OrderChange: orderChange}); err != nil {
			return nil, err
		}
		original := draftordersteps.FindOrderChangeAction(orderChange, in.ActionID)
		var price any
		if hasOptions {
			price = g(rec(g(option, "calculated_price")), "calculated_amount")
		} else {
			price = g(in.Data, "custom_amount")
		}
		action := dal.NewRecord()
		action.Set("id", g(original, "id"))
		action.Set("amount", price)
		setIfPresent(action, in.Data, "internal_note")
		method := dal.NewRecord()
		method.Set("id", g(original, "reference_id"))
		method.Set("amount", price)
		method.Set("is_custom_amount", !hasOptions)
		setIfPresent(method, in.Data, "metadata")
		err = composer.Parallelize(
			func() error {
				_, err := ordersteps.UpdateOrderChangeActionsStep.Run(e, []*dal.Record{action})
				return err
			},
			func() error {
				_, err := ordersteps.UpdateOrderShippingMethodsStep.Run(e, []*dal.Record{method})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		if err := computeIfPromotions(e, order, in.OrderID); err != nil {
			return nil, err
		}
		if err := releaseLock(e, in.OrderID); err != nil {
			return nil, err
		}
		return preview(e, in.OrderID)
	})

// RemoveDraftOrderActionShippingMethodWorkflow mirrors removeDraftOrderActionShippingMethodWorkflow.
var RemoveDraftOrderActionShippingMethodWorkflow = composer.CreateWorkflow("remove-draft-order-action-shipping-method",
	func(e *composer.Execution, in DraftOrderActionInput) (*dal.Record, error) {
		if err := acquireLock(e, in.OrderID); err != nil {
			return nil, err
		}
		order, err := remoteOne(e, "order-query", "orders", in.OrderID, DraftOrderFieldsForRefreshSteps)
		if err != nil {
			return nil, err
		}
		orderChange, err := draftActiveChange(e, in.OrderID, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if err := validateDraftChange(e, order, orderChange); err != nil {
			return nil, err
		}
		if _, err := draftordersteps.ValidateDraftOrderShippingMethodActionStep.Run(e, draftordersteps.ActionValidationInput{OrderID: in.OrderID, ActionID: in.ActionID, OrderChange: orderChange}); err != nil {
			return nil, err
		}
		action := draftordersteps.FindOrderChangeAction(orderChange, in.ActionID)
		err = composer.Parallelize(
			func() error {
				_, err := ordersteps.DeleteOrderChangeActionsStep.Run(e, []any{g(action, "id")})
				return err
			},
			func() error {
				_, err := ordersteps.DeleteOrderShippingMethodsStep.Run(e, []any{g(action, "reference_id")})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		if err := computeIfPromotions(e, order, in.OrderID); err != nil {
			return nil, err
		}
		if err := releaseLock(e, in.OrderID); err != nil {
			return nil, err
		}
		return preview(e, in.OrderID)
	})

// UpdateDraftOrderShippingMethodWorkflow mirrors updateDraftOrderShippingMethodWorkflow.
var UpdateDraftOrderShippingMethodWorkflow = composer.CreateWorkflow("update-draft-order-shipping-method",
	func(e *composer.Execution, in DraftOrderActionInput) (*dal.Record, error) {
		if err := acquireLock(e, in.OrderID); err != nil {
			return nil, err
		}
		order, err := remoteOne(e, "order-query", "orders", in.OrderID, []string{"id", "status", "is_draft_order"})
		if err != nil {
			return nil, err
		}
		orderChange, err := draftActiveChange(e, in.OrderID, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if err := validateDraftChange(e, order, orderChange); err != nil {
			return nil, err
		}
		methodID := g(in.Data, "shipping_method_id")
		res, err := draftordersteps.UpdateDraftOrderShippingMethodStep.Run(e, draftordersteps.UpdateDraftOrderShippingMethodInput{
			OrderID: in.OrderID, ShippingMethodID: methodID, ShippingOptionID: g(in.Data, "shipping_option_id"), Amount: g(in.Data, "custom_amount"),
		})
		if err != nil {
			return nil, err
		}
		if _, err := orderworkflows.UpdateOrderTaxLinesWorkflow.RunAsStep(e, orderworkflows.UpdateOrderTaxLinesInput{OrderID: g(order, "id"), ShippingMethodIDs: []any{methodID}}); err != nil {
			return nil, err
		}
		a := dal.NewRecord()
		a.Set("order_change_id", g(orderChange, "id"))
		a.Set("reference", "order_shipping_method")
		a.Set("reference_id", methodID)
		a.Set("order_id", g(order, "id"))
		a.Set("version", g(orderChange, "version"))
		a.Set("action", "SHIPPING_UPDATE")
		setIfPresent(a, in.Data, "internal_note")
		d := dal.NewRecord()
		d.Set("old_shipping_option_id", g(res.Before, "shipping_option_id"))
		d.Set("new_shipping_option_id", g(res.After, "shipping_option_id"))
		d.Set("old_amount", g(res.Before, "amount"))
		d.Set("new_amount", g(res.After, "amount"))
		a.Set("details", d)
		if _, err := orderworkflows.CreateOrderChangeActionsWorkflow.RunAsStep(e, []*dal.Record{a}); err != nil {
			return nil, err
		}
		refetched, err := remoteOne(e, "refetched-order-query", "orders", in.OrderID, DraftOrderFieldsForRefreshSteps)
		if err != nil {
			return nil, err
		}
		if err := computeIfPromotions(e, refetched, in.OrderID); err != nil {
			return nil, err
		}
		if err := releaseLock(e, in.OrderID); err != nil {
			return nil, err
		}
		return preview(e, in.OrderID)
	})

// RemoveDraftOrderShippingMethodWorkflow mirrors removeDraftOrderShippingMethodWorkflow.
var RemoveDraftOrderShippingMethodWorkflow = composer.CreateWorkflow("remove-draft-order-shipping-method",
	func(e *composer.Execution, in DraftOrderActionInput) (*dal.Record, error) {
		if err := acquireLock(e, in.OrderID); err != nil {
			return nil, err
		}
		order, err := remoteOne(e, "order-query", "orders", in.OrderID, DraftOrderFieldsForRefreshSteps)
		if err != nil {
			return nil, err
		}
		orderChange, err := draftActiveChange(e, in.OrderID, []string{"id", "status", "version"})
		if err != nil {
			return nil, err
		}
		if err := validateDraftChange(e, order, orderChange); err != nil {
			return nil, err
		}
		a := dal.NewRecord()
		a.Set("action", "SHIPPING_REMOVE")
		a.Set("reference", "order_shipping_method")
		a.Set("order_change_id", g(orderChange, "id"))
		a.Set("reference_id", g(in.Data, "shipping_method_id"))
		a.Set("order_id", g(order, "id"))
		if _, err := orderworkflows.CreateOrderChangeActionsWorkflow.RunAsStep(e, []*dal.Record{a}); err != nil {
			return nil, err
		}
		if err := computeIfPromotions(e, order, in.OrderID); err != nil {
			return nil, err
		}
		if err := releaseLock(e, in.OrderID); err != nil {
			return nil, err
		}
		return preview(e, g(order, "id"))
	})

// DraftOrderEditUserInput mirrors the request/confirm workflow inputs.
type DraftOrderEditUserInput struct {
	OrderID any
	UserID  any
}

// RequestDraftOrderEditWorkflow mirrors requestDraftOrderEditWorkflow.
var RequestDraftOrderEditWorkflow = composer.CreateWorkflow("request-draft-order-edit",
	func(e *composer.Execution, in DraftOrderEditUserInput) (*dal.Record, error) {
		if err := acquireLock(e, in.OrderID); err != nil {
			return nil, err
		}
		order, err := remoteOne(e, "order-query", "orders", in.OrderID, []string{"id", "version", "status", "is_draft_order", "canceled_at"})
		if err != nil {
			return nil, err
		}
		orderChange, err := draftActiveChange(e, in.OrderID, []string{"id", "canceled_at"})
		if err != nil {
			return nil, err
		}
		if err := validateDraftChange(e, order, orderChange); err != nil {
			return nil, err
		}
		u := dal.NewRecord()
		u.Set("id", g(orderChange, "id"))
		u.Set("status", "requested")
		u.Set("requested_at", time.Now())
		if in.UserID != nil {
			u.Set("requested_by", in.UserID)
		}
		if _, err := ordersteps.UpdateOrderChangesStep.Run(e, []*dal.Record{u}); err != nil {
			return nil, err
		}
		pc := dal.NewRecord()
		pc.Set("order_id", g(order, "id"))
		if _, err := orderworkflows.CreateOrUpdateOrderPaymentCollectionWorkflow.RunAsStep(e, pc); err != nil {
			return nil, err
		}
		if err := releaseLock(e, in.OrderID); err != nil {
			return nil, err
		}
		return preview(e, g(order, "id"))
	})

// ConfirmDraftOrderEditWorkflow mirrors confirmDraftOrderEditWorkflow.
var ConfirmDraftOrderEditWorkflow = composer.CreateWorkflow("confirm-draft-order-edit",
	func(e *composer.Execution, in DraftOrderEditUserInput) (*dal.Record, error) {
		if err := acquireLock(e, in.OrderID); err != nil {
			return nil, err
		}
		order, err := remoteOne(e, "order-query", "orders", in.OrderID, []string{"id", "status", "is_draft_order", "version", "canceled_at",
			"items.id", "items.title", "items.variant_title", "items.variant_sku", "items.variant_barcode", "shipping_address.*"})
		if err != nil {
			return nil, err
		}
		orderChange, err := draftActiveChange(e, in.OrderID, []string{"id", "status", "actions.id", "actions.order_id", "actions.return_id",
			"actions.action", "actions.details", "actions.reference", "actions.reference_id", "actions.internal_note"})
		if err != nil {
			return nil, err
		}
		if err := validateDraftChange(e, order, orderChange); err != nil {
			return nil, err
		}
		if _, err := ordersteps.ConfirmOrderChangesStep.Run(e, ordersteps.ConfirmOrderChangesInput{OrderID: g(order, "id"), Changes: []*dal.Record{orderChange}, ConfirmedBy: in.UserID}); err != nil {
			return nil, err
		}
		itemActions := false
		for _, a := range recs(g(orderChange, "actions")) {
			switch g(a, "action") {
			case "ITEM_ADD", "ITEM_REMOVE", "ITEM_UPDATE":
				itemActions = true
			}
		}
		if itemActions {
			if _, err := RefreshConfirmedDraftOrderShippingMethodsWorkflow.RunAsStep(e, g(order, "id")); err != nil {
				return nil, err
			}
		}
		pc := dal.NewRecord()
		pc.Set("order_id", g(order, "id"))
		if _, err := orderworkflows.CreateOrUpdateOrderPaymentCollectionWorkflow.RunAsStep(e, pc); err != nil {
			return nil, err
		}
		orderPreview, err := preview(e, g(order, "id"))
		if err != nil {
			return nil, err
		}
		if err := releaseLock(e, in.OrderID); err != nil {
			return nil, err
		}
		return orderPreview, nil
	})

// CancelDraftOrderEditWorkflow mirrors cancelDraftOrderEditWorkflow.
var CancelDraftOrderEditWorkflow = composer.CreateWorkflow("cancel-draft-order-edit",
	func(e *composer.Execution, orderID any) (any, error) {
		if err := acquireLock(e, orderID); err != nil {
			return nil, err
		}
		order, err := remoteOne(e, "order-query", "orders", orderID, append([]string{"version"}, DraftOrderFieldsForRefreshSteps...))
		if err != nil {
			return nil, err
		}
		orderChange, err := draftActiveChange(e, orderID, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if err := validateDraftChange(e, order, orderChange); err != nil {
			return nil, err
		}
		actions := recs(g(orderChange, "actions"))
		var toRemove, toRestore []string
		for _, a := range actions {
			d := rec(g(a, "details"))
			switch g(a, "action") {
			case "PROMOTION_ADD":
				if c, ok := g(d, "added_code").(string); ok && c != "" {
					toRemove = append(toRemove, c)
				}
			case "PROMOTION_REMOVE":
				if c, ok := g(d, "removed_code").(string); ok && c != "" {
					toRestore = append(toRestore, c)
				}
			}
		}
		var codes []string
		seen := map[string]bool{}
		add := func(c string) {
			if !seen[c] {
				seen[c] = true
				codes = append(codes, c)
			}
		}
		for _, c := range promotionCodes(order) {
			add(c)
		}
		for _, c := range toRemove {
			if seen[c] {
				delete(seen, c)
				for i, x := range codes {
					if x == c {
						codes = append(codes[:i], codes[i+1:]...)
						break
					}
				}
			}
		}
		for _, c := range toRestore {
			add(c)
		}
		if _, err := draftordersteps.UpdateDraftOrderPromotionsStep.Run(e, draftordersteps.UpdateDraftOrderPromotionsInput{ID: orderID, PromoCodes: codes, Action: cartsteps.PromotionActionReplace}); err != nil {
			return nil, err
		}
		shippingToRemove := []any{}
		var shippingToRestore []draftordersteps.ShippingMethodRestore
		for _, a := range actions {
			switch g(a, "action") {
			case "SHIPPING_ADD":
				shippingToRemove = append(shippingToRemove, g(a, "reference_id"))
			case "SHIPPING_UPDATE":
				d := rec(g(a, "details"))
				before, after := dal.NewRecord(), dal.NewRecord()
				for _, pair := range [][3]string{{"shipping_option_id", "old_shipping_option_id", "new_shipping_option_id"}, {"amount", "old_amount", "new_amount"}} {
					if v := g(d, pair[1]); v != nil {
						before.Set(pair[0], v)
					}
					if v := g(d, pair[2]); v != nil {
						after.Set(pair[0], v)
					}
				}
				shippingToRestore = append(shippingToRestore, draftordersteps.ShippingMethodRestore{ID: g(a, "reference_id"), Before: before, After: after})
			}
		}
		err = composer.Parallelize(
			func() error {
				_, err := ordersteps.DeleteOrderChangesStep.Run(e, []any{g(orderChange, "id")})
				return err
			},
			func() error {
				_, err := ordersteps.DeleteOrderShippingMethodsStep.Run(e, shippingToRemove)
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		if len(shippingToRestore) > 0 {
			if _, err := draftordersteps.RestoreDraftOrderShippingMethodsStep.Run(e, shippingToRestore); err != nil {
				return nil, err
			}
		}
		return nil, releaseLock(e, orderID)
	})
