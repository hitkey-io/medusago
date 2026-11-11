// Package workflows mirrors @medusajs/core-flows/src/draft-order/workflows.
package workflows

import (
	"fmt"

	cartsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	cartutils "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	draftordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/draft-order/steps"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	orderworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func g(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func rec(v any) *dal.Record {
	switch t := v.(type) {
	case *dal.Record:
		return t
	case []any:
		if len(t) > 0 {
			r, _ := t[0].(*dal.Record)
			return r
		}
	case []*dal.Record:
		if len(t) > 0 {
			return t[0]
		}
	}
	return nil
}

func recs(v any) []*dal.Record {
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

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	case int:
		return t != 0
	}
	return true
}

func remoteOne(e *composer.Execution, name, entry string, id any, fields []string) (*dal.Record, error) {
	v := dal.NewRecord()
	v.Set("id", id)
	list := false
	res, err := commonsteps.UseRemoteQueryStep.Config(name, composer.StepOptions{}).Run(e, commonsteps.RemoteStepInput{EntryPoint: entry, Fields: fields, Variables: v, List: &list, ThrowIfKeyNotFound: true})
	if err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, nil
	}
	return rec(res.Data[0]), nil
}

func graphOne(e *composer.Execution, name, entity string, filters *dal.Record, fields []string, throw bool) (*dal.Record, error) {
	isList := false
	res, err := commonsteps.UseQueryGraphStep.Config(name, composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
		GraphInput: query.GraphInput{Entity: entity, Fields: fields, Filters: filters},
		Options:    joiner.Options{ThrowIfKeyNotFound: throw},
		IsList:     &isList,
	})
	if err != nil {
		return nil, err
	}
	return rec(res.Data), nil
}

func graphList(e *composer.Execution, name, entity string, filters *dal.Record, fields []string) ([]*dal.Record, error) {
	res, err := commonsteps.UseQueryGraphStep.Config(name, composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
		GraphInput: query.GraphInput{Entity: entity, Fields: fields, Filters: filters},
	})
	if err != nil {
		return nil, err
	}
	return recs(res.Data), nil
}

func notAllowedCalculated() error {
	return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Calculated shipping option prices are not supported yet")
}

var shippingOptionFieldsForPriceCalculation = []string{"id", "name", "price_type", "service_zone_id", "service_zone.fulfillment_set_id",
	"service_zone.fulfillment_set.type", "service_zone.fulfillment_set.location.*", "service_zone.fulfillment_set.location.address.*",
	"shipping_profile_id", "provider_id", "data", "type.id", "type.label", "type.description", "type.code", "provider.id", "provider.is_enabled",
	"rules.attribute", "rules.value", "rules.operator"}

// RefreshPendingDraftOrderShippingMethodsWorkflow mirrors
// refreshPendingDraftOrderShippingMethodsWorkflow (calculated prices are not
// supported yet).
var RefreshPendingDraftOrderShippingMethodsWorkflow = composer.CreateWorkflow("refresh-pending-draft-order-shipping-methods",
	func(e *composer.Execution, orderID any) (any, error) {
		f := dal.NewRecord()
		f.Set("order_id", orderID)
		f.Set("status", []any{"pending", "requested"})
		change, err := graphOne(e, "refresh-order-change-query", "order_change", f, []string{"id", "actions.id", "actions.action", "actions.reference_id", "order.id", "order.shipping_address.*"}, false)
		if err != nil {
			return nil, err
		}
		pending := []any{}
		for _, a := range recs(g(change, "actions")) {
			if g(a, "action") == "SHIPPING_ADD" && truthy(g(a, "reference_id")) {
				pending = append(pending, g(a, "reference_id"))
			}
		}
		mf := dal.NewRecord()
		mf.Set("id", pending)
		mf.Set("is_custom_amount", false)
		methods, err := graphList(e, "refresh-pending-methods-query", "order_shipping_method", mf, []string{"id", "shipping_option_id"})
		if err != nil {
			return nil, err
		}
		hookIn := dal.NewRecord()
		hookIn.Set("input", orderID)
		if err := composer.CreateHook(e, "setCalculatedShippingPricingContext", hookIn); err != nil {
			return nil, err
		}
		if len(methods) == 0 {
			return nil, nil
		}
		var optionIDs []any
		for _, m := range methods {
			if v := g(m, "shipping_option_id"); truthy(v) {
				optionIDs = append(optionIDs, v)
			}
		}
		of := dal.NewRecord()
		of.Set("id", optionIDs)
		options, err := graphList(e, "refresh-shipping-options-query", "shipping_option", of, shippingOptionFieldsForPriceCalculation)
		if err != nil {
			return nil, err
		}
		for _, o := range options {
			if g(o, "price_type") == "calculated" {
				return nil, notAllowedCalculated()
			}
		}
		if _, err := ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(orderID)); err != nil {
			return nil, err
		}
		return nil, nil
	})

// RefreshConfirmedDraftOrderShippingMethodsWorkflow mirrors
// refreshConfirmedDraftOrderShippingMethodsWorkflow (calculated prices are
// not supported yet).
var RefreshConfirmedDraftOrderShippingMethodsWorkflow = composer.CreateWorkflow("refresh-confirmed-draft-order-shipping-methods",
	func(e *composer.Execution, orderID any) (any, error) {
		f := dal.NewRecord()
		f.Set("id", orderID)
		order, err := graphOne(e, "refresh-confirmed-order-query", "order", f, []string{"id", "shipping_address.*", "items.*", "items.variant.*",
			"items.variant.product.shipping_profile.id", "shipping_methods.id", "shipping_methods.is_custom_amount", "shipping_methods.shipping_option_id"}, true)
		if err != nil {
			return nil, err
		}
		optionIDs := []any{}
		for _, m := range recs(g(order, "shipping_methods")) {
			if truthy(g(m, "is_custom_amount")) {
				continue
			}
			if v := g(m, "shipping_option_id"); truthy(v) {
				optionIDs = append(optionIDs, v)
			}
		}
		of := dal.NewRecord()
		of.Set("id", optionIDs)
		options, err := graphList(e, "refresh-confirmed-options-query", "shipping_option", of, shippingOptionFieldsForPriceCalculation)
		if err != nil {
			return nil, err
		}
		hookIn := dal.NewRecord()
		hookIn.Set("input", orderID)
		if err := composer.CreateHook(e, "setCalculatedShippingPricingContext", hookIn); err != nil {
			return nil, err
		}
		for _, o := range options {
			if g(o, "price_type") == "calculated" {
				return nil, notAllowedCalculated()
			}
		}
		return nil, nil
	})

// UpdateDraftOrderWorkflow mirrors updateDraftOrderWorkflow (locks are not
// ported).
var UpdateDraftOrderWorkflow = composer.CreateWorkflow("update-draft-order",
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		id := g(in, "id")
		order, err := remoteOne(e, "order-query", "orders", id, []string{"id", "customer_id", "status", "is_draft_order", "sales_channel_id", "email",
			"customer_id", "locale", "shipping_address.*", "billing_address.*", "shipping_methods.id", "shipping_methods.name",
			"shipping_methods.shipping_option_id", "metadata"})
		if err != nil {
			return nil, err
		}
		if _, err := draftordersteps.ValidateDraftOrderStep.Run(e, order); err != nil {
			return nil, err
		}
		update := dal.NewRecord()
		for _, k := range in.Keys() {
			v, _ := in.Get(k)
			update.Set(k, v)
		}
		for _, k := range []string{"shipping_address", "billing_address"} {
			if a := rec(g(in, k)); a != nil {
				addr := dal.NewRecord()
				if prev := rec(g(order, k)); prev != nil {
					for _, kk := range prev.Keys() {
						v, _ := prev.Get(kk)
						addr.Set(kk, v)
					}
				}
				for _, kk := range a.Keys() {
					v, _ := a.Get(kk)
					addr.Set(kk, v)
				}
				addr.Delete("id")
				update.Set(k, addr)
			}
		}
		updated, err := draftordersteps.UpdateDraftOrderStep.Run(e, draftordersteps.UpdateDraftOrderStepInput{Order: order, Input: update})
		if err != nil {
			return nil, err
		}
		var changes []*dal.Record
		addChange := func(typ string, prev, next any) {
			d := dal.NewRecord()
			d.Set("type", typ)
			d.Set("old", prev)
			d.Set("new", next)
			c := dal.NewRecord()
			c.Set("change_type", "update_order")
			c.Set("order_id", id)
			c.Set("created_by", g(in, "user_id"))
			c.Set("confirmed_by", g(in, "user_id"))
			c.Set("details", d)
			changes = append(changes, c)
		}
		for _, k := range []string{"shipping_address", "billing_address", "customer_id", "email", "sales_channel_id", "metadata"} {
			if truthy(g(in, k)) {
				addChange(k, g(order, k), g(updated, k))
			}
		}
		if truthy(g(in, "locale")) && g(in, "locale") != g(order, "locale") {
			addChange("locale", g(order, "locale"), g(updated, "locale"))
		}
		if changes == nil {
			changes = []*dal.Record{}
		}
		if _, err := ordersteps.RegisterOrderChangesStep.Run(e, changes); err != nil {
			return nil, err
		}
		if truthy(g(in, "shipping_address")) {
			err := composer.Parallelize(
				func() error { _, err := RefreshPendingDraftOrderShippingMethodsWorkflow.RunAsStep(e, id); return err },
				func() error { _, err := RefreshConfirmedDraftOrderShippingMethodsWorkflow.RunAsStep(e, id); return err },
			)
			if err != nil {
				return nil, err
			}
		}
		if truthy(g(in, "shipping_address")) || truthy(g(in, "billing_address")) {
			if _, err := orderworkflows.UpdateOrderTaxLinesWorkflow.RunAsStep(e, orderworkflows.UpdateOrderTaxLinesInput{OrderID: id}); err != nil {
				return nil, err
			}
		}
		if truthy(g(in, "locale")) && g(in, "locale") != g(order, "locale") {
			if err := orderworkflows.RetranslateOrder(e, fmt.Sprint(id), fmt.Sprint(g(in, "locale")), recs(g(order, "shipping_methods")), true); err != nil {
				return nil, err
			}
		}
		ev := dal.NewRecord()
		ev.Set("id", id)
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "order.updated", Data: ev}); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(id))
	})

var validateDraftOrdersStep = composer.CreateStep("validate-draft-orders",
	func(ctx composer.StepExecutionContext, orders []*dal.Record) (composer.StepResponse[any], error) {
		for _, o := range orders {
			if g(o, "status") != "draft" || !truthy(g(o, "is_draft_order")) {
				return composer.StepResponse[any]{}, fmt.Errorf("One or more orders are not draft")
			}
		}
		for _, o := range orders {
			if truthy(g(o, "deleted_at")) {
				return composer.StepResponse[any]{}, fmt.Errorf("One or more orders are already deleted")
			}
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// DeleteDraftOrdersWorkflow mirrors deleteDraftOrdersWorkflow.
var DeleteDraftOrdersWorkflow = composer.CreateWorkflow("delete-draft-order",
	func(e *composer.Execution, ids []any) (any, error) {
		f := dal.NewRecord()
		f.Set("id", ids)
		res, err := commonsteps.UseQueryGraphStep.Config("get-draft-order", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "orders", Fields: []string{"id", "status", "is_draft_order", "deleted_at"}, Filters: f},
			Options:    joiner.Options{ThrowIfKeyNotFound: true},
		})
		if err != nil {
			return nil, err
		}
		if _, err := validateDraftOrdersStep.Run(e, recs(res.Data)); err != nil {
			return nil, err
		}
		o := dal.NewRecord()
		o.Set("order_id", ids)
		link := dal.NewRecord()
		link.Set(utilsmodulessdk.ORDER, o)
		if _, err := commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{link}); err != nil {
			return nil, err
		}
		_, err = draftordersteps.DeleteDraftOrdersStep.Run(e, ids)
		return nil, err
	})

// ConvertDraftOrderWorkflow mirrors convertDraftOrderWorkflow (locks are not
// ported).
var ConvertDraftOrderWorkflow = composer.CreateWorkflow("convert-draft-order",
	func(e *composer.Execution, id any) (*dal.Record, error) {
		order, err := remoteOne(e, "order-query", "orders", id, []string{"id", "status", "is_draft_order"})
		if err != nil {
			return nil, err
		}
		if _, err := draftordersteps.ValidateDraftOrderStep.Run(e, order); err != nil {
			return nil, err
		}
		if _, err := orderworkflows.UpdateOrderTaxLinesWorkflow.RunAsStep(e, orderworkflows.UpdateOrderTaxLinesInput{OrderID: id}); err != nil {
			return nil, err
		}
		orderItems, err := remoteOne(e, "order-items-query", "order", id, orderworkflows.RequiredOrderFieldsForInventoryConfirmation)
		if err != nil {
			return nil, err
		}
		cf := dal.NewRecord()
		cf.Set("order_id", id)
		cv := dal.NewRecord()
		cv.Set("filters", cf)
		cres, err := commonsteps.UseRemoteQueryStep.Config("order-changes-query", composer.StepOptions{}).Run(e, commonsteps.RemoteStepInput{
			EntryPoint: "order_change", Fields: []string{"id", "actions.action", "actions.details"}, Variables: cv,
		})
		if err != nil {
			return nil, err
		}
		backorder := map[any]bool{}
		for _, c := range recs(cres.Data) {
			for _, a := range recs(g(c, "actions")) {
				d := rec(g(a, "details"))
				if g(a, "action") == "ITEM_ADD" && truthy(g(d, "allow_backorder")) {
					backorder[g(d, "reference_id")] = true
				}
			}
		}
		var items, variants []*dal.Record
		for _, oi := range recs(g(orderItems, "items")) {
			it := dal.NewRecord()
			it.Set("variant_id", g(rec(g(oi, "variant")), "id"))
			it.Set("quantity", g(oi, "quantity"))
			it.Set("id", g(oi, "id"))
			it.Set("allow_backorder", backorder[g(oi, "id")])
			items = append(items, it)
			if v := rec(g(oi, "variant")); v != nil {
				variants = append(variants, v)
			}
		}
		channelID, _ := g(orderItems, "sales_channel_id").(string)
		inv, err := cartutils.PrepareConfirmInventoryInput(cartutils.ConfirmVariantInventoryInput{SalesChannelID: channelID, Variants: variants, Items: items})
		if err != nil {
			return nil, err
		}
		reservations, err := cartsteps.ReserveInventoryStep.Run(e, inv)
		if err != nil {
			return nil, err
		}
		events := []any{}
		for _, r := range reservations {
			ev := dal.NewRecord()
			ev.Set("id", g(r, "id"))
			ev.Set("order_id", g(order, "id"))
			events = append(events, ev)
		}
		if _, err := commonsteps.EmitEventStep.Config("emit-reservation-item-created", composer.StepOptions{}).Run(e, commonsteps.EmitEventStepInput{EventName: "reservation-item.created", Data: events}); err != nil {
			return nil, err
		}
		updated, err := draftordersteps.ConvertDraftOrderStep.Run(e, id)
		if err != nil {
			return nil, err
		}
		placed := dal.NewRecord()
		placed.Set("id", g(updated, "id"))
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "order.placed", Data: placed}); err != nil {
			return nil, err
		}
		return updated, nil
	})
