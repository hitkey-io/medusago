package workflows

import (
	commonutils "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/utils"
	"strings"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	paymentsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

var validateOrderCreditLinesStep = composer.CreateStep("validate-order-credit-lines",
	func(ctx composer.StepExecutionContext, in struct {
		Order       *dal.Record
		CreditLines []*dal.Record
	}) (composer.StepResponse[any], error) {
		pending := g(rec(g(in.Order, "summary")), "pending_difference")
		if pending == nil {
			pending = 0
		}
		amount := totals.Convert(0)
		for _, cl := range in.CreditLines {
			amount = totals.Add(amount, g(cl, "amount"))
		}
		fail := func(msg string) (composer.StepResponse[any], error) {
			return composer.StepResponse[any]{}, invalidData(msg)
		}
		if totals.Eq(pending, 0) {
			return fail("Can only create credit lines if the order has a positive or negative pending difference")
		}
		if totals.Gt(pending, 0) && totals.Lt(amount, 0) {
			return fail("Can only create positive credit lines if the order has a positive pending difference")
		}
		if totals.Lt(pending, 0) && totals.Gt(amount, 0) {
			return fail("Can only create negative credit lines if the order has a negative pending difference")
		}
		if totals.Lt(pending, 0) && totals.Gt(totals.Mult(amount, -1), totals.Mult(pending, -1)) {
			return fail("Cannot create more negative credit lines with amount more than the pending difference")
		}
		if totals.Gt(pending, 0) && totals.Gt(amount, pending) {
			return fail("Cannot create more positive credit lines with amount more than the pending difference")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

var CreateOrderChangeActionsWorkflow = composer.CreateWorkflow("create-order-change-actions",
	func(e *composer.Execution, in []*dal.Record) ([]*dal.Record, error) {
		return ordersteps.AddOrderActionsStep.Run(e, in)
	})

// CreateOrderCreditLinesWorkflow mirrors createOrderCreditLinesWorkflow.
var CreateOrderCreditLinesWorkflow = composer.CreateWorkflow("create-order-credit-lines",
	func(e *composer.Execution, in struct {
		ID          any
		CreditLines []*dal.Record
	}) (any, error) {
		f := dal.NewRecord()
		f.Set("id", in.ID)
		res, err := commonsteps.UseQueryGraphStep.Config("get-order", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "orders", Fields: []string{"id", "status", "summary", "total"}, Filters: f},
			Options:    joiner.Options{ThrowIfKeyNotFound: true},
		})
		if err != nil {
			return nil, err
		}
		var order *dal.Record
		if list := recs(res.Data); len(list) > 0 {
			order = list[0]
		}
		if _, err := validateOrderCreditLinesStep.Run(e, struct {
			Order       *dal.Record
			CreditLines []*dal.Record
		}{order, in.CreditLines}); err != nil {
			return nil, err
		}
		changeIn := dal.NewRecord()
		changeIn.Set("change_type", "credit_line")
		changeIn.Set("order_id", in.ID)
		change, err := ordersteps.CreateOrderChangeStep.Run(e, changeIn)
		if err != nil {
			return nil, err
		}
		var actions []*dal.Record
		for _, cl := range in.CreditLines {
			a := dal.NewRecord()
			a.Set("order_change_id", g(change, "id"))
			a.Set("order_id", g(order, "id"))
			a.Set("version", g(change, "version"))
			a.Set("action", "CREDIT_LINE_ADD")
			a.Set("reference", g(cl, "reference"))
			a.Set("reference_id", g(cl, "reference_id"))
			a.Set("amount", g(cl, "amount"))
			actions = append(actions, a)
		}
		if _, err := CreateOrderChangeActionsWorkflow.RunAsStep(e, actions); err != nil {
			return nil, err
		}
		cf := dal.NewRecord()
		cf.Set("order_id", in.ID)
		cf.Set("status", []any{"pending"})
		changes, err := commonsteps.UseQueryGraphStep.Config("order-change-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "order_change", Fields: []string{"id", "status", "version", "change_type", "actions.id", "actions.order_id", "actions.action",
				"actions.details", "actions.reference", "actions.reference_id", "actions.internal_note"}, Filters: cf},
		})
		if err != nil {
			return nil, err
		}
		var orderChange *dal.Record
		if list := recs(changes.Data); len(list) > 0 {
			orderChange = list[0]
		}
		if _, err := ordersteps.ConfirmOrderChangesStep.Run(e, ordersteps.ConfirmOrderChangesInput{OrderID: g(order, "id"), Changes: []*dal.Record{orderChange}}); err != nil {
			return nil, err
		}
		hook := dal.NewRecord()
		hook.Set("order_id", in.ID)
		return nil, composer.CreateHook(e, "creditLinesCreated", hook)
	})

var validatePresenceOfOrder = composer.CreateStep("validate-presence-of",
	func(ctx composer.StepExecutionContext, in struct {
		Entity *dal.Record
		Fields []string
	}) (composer.StepResponse[any], error) {
		var missing []string
		for _, f := range in.Fields {
			if !truthy(g(in.Entity, f)) {
				missing = append(missing, f)
			}
		}
		if len(missing) > 0 {
			return composer.StepResponse[any]{}, invalidData("Field(s) are required to have value to continue - " + strings.Join(missing, ", "))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// ListShippingOptionsForOrderWorkflow mirrors listShippingOptionsForOrderWorkflow.
var ListShippingOptionsForOrderWorkflow = composer.CreateWorkflow("list-shipping-options-for-order",
	func(e *composer.Execution, orderID string) ([]any, error) {
		f := dal.NewRecord()
		f.Set("id", orderID)
		res, err := commonsteps.UseQueryGraphStep.Config("get-order", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "order", Fields: []string{"id", "sales_channel_id", "region_id", "shipping_address.city", "shipping_address.country_code",
				"shipping_address.province", "shipping_address.postal_code", "items.*", "items.variant.manage_inventory", "items.variant.inventory_items.inventory_item_id",
				"items.variant.inventory_items.inventory.requires_shipping", "items.variant.inventory_items.inventory.location_levels.*"}, Filters: f},
			Options: joiner.Options{ThrowIfKeyNotFound: true},
		})
		if err != nil {
			return nil, err
		}
		var order *dal.Record
		if list := recs(res.Data); len(list) > 0 {
			order = list[0]
		}
		if _, err := validatePresenceOfOrder.Run(e, struct {
			Entity *dal.Record
			Fields []string
		}{order, []string{"sales_channel_id", "region_id"}}); err != nil {
			return nil, err
		}
		sf := dal.NewRecord()
		sf.Set("id", g(order, "sales_channel_id"))
		sc, err := commonsteps.UseQueryGraphStep.Config("sales_channels-fulfillment-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "sales_channels", Fields: []string{"id", "stock_locations.fulfillment_sets.id", "stock_locations.id", "stock_locations.name", "stock_locations.address.*"}, Filters: sf},
			Options:    joiner.Options{Cache: caching.WithTags(true, commonutils.SalesChannelStockLocationCacheTags...)},
		})
		if err != nil {
			return nil, err
		}
		var setIDs []any
		seen := map[any]bool{}
		if list := recs(sc.Data); len(list) > 0 {
			for _, sl := range recs(g(list[0], "stock_locations")) {
				for _, fs := range recs(g(sl, "fulfillment_sets")) {
					if !seen[g(fs, "id")] {
						seen[g(fs, "id")] = true
						setIDs = append(setIDs, g(fs, "id"))
					}
				}
			}
		}
		if setIDs == nil {
			setIDs = []any{}
		}
		sa := rec(g(order, "shipping_address"))
		addr := dal.NewRecord()
		addr.Set("country_code", g(sa, "country_code"))
		addr.Set("province_code", g(sa, "province"))
		addr.Set("city", g(sa, "city"))
		addr.Set("postal_expression", g(sa, "postal_code"))
		of := dal.NewRecord()
		of.Set("fulfillment_set_id", setIDs)
		of.Set("address", addr)
		so, err := commonsteps.UseQueryGraphStep.Config("shipping-options-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "shipping_option", Fields: []string{"id", "name", "price_type", "service_zone_id", "shipping_profile_id", "provider_id", "data",
				"service_zone.fulfillment_set_id", "service_zone.fulfillment_set.type", "service_zone.fulfillment_set.location.id", "service_zone.fulfillment_set.location.name",
				"service_zone.fulfillment_set.location.address.*", "type.id", "type.label", "type.description", "type.code", "provider.id", "provider.is_enabled",
				"rules.attribute", "rules.value", "rules.operator"}, Filters: of},
		})
		if err != nil {
			return nil, err
		}
		out := []any{}
		for _, opt := range recs(so.Data) {
			locationID := g(rec(g(rec(g(rec(g(opt, "service_zone")), "fulfillment_set")), "location")), "id")
			insufficient := false
			for _, item := range recs(g(order, "items")) {
				variant := rec(g(item, "variant"))
				if !truthy(g(variant, "manage_inventory")) {
					continue
				}
				for _, ii := range recs(g(variant, "inventory_items")) {
					inv := rec(g(ii, "inventory"))
					if !truthy(g(inv, "requires_shipping")) {
						continue
					}
					var level *dal.Record
					for _, l := range recs(g(inv, "location_levels")) {
						if g(l, "location_id") == locationID {
							level = l
							break
						}
					}
					if level == nil || totals.Lt(g(level, "available_quantity"), g(item, "quantity")) {
						insufficient = true
					}
				}
			}
			r := dal.NewRecord()
			for _, k := range opt.Keys() {
				v, _ := opt.Get(k)
				r.Set(k, v)
			}
			r.Set("insufficient_inventory", insufficient)
			out = append(out, r)
		}
		return out, nil
	})

var validatePendingAuthorizationStep = composer.CreateStep("validate-pending-authorization-step",
	func(ctx composer.StepExecutionContext, s *dal.Record) (composer.StepResponse[any], error) {
		if g(s, "status") != "pending_authorization" {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Payment session "+str(g(s, "id"))+" is not in pending_authorization status")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

func str(v any) string {
	s, _ := v.(string)
	return s
}

// AuthorizePaymentSessionForOrderWorkflow mirrors authorizePaymentSessionForOrderWorkflow.
var AuthorizePaymentSessionForOrderWorkflow = composer.CreateWorkflow("authorize-payment-session-for-order",
	func(e *composer.Execution, sessionID string) (*dal.Record, error) {
		session, err := queryOne(e, "get-payment-session", "payment_session", sessionID, []string{"id", "status", "payment_collection_id"})
		if err != nil {
			return nil, err
		}
		check := dal.NewRecord()
		check.Set("id", sessionID)
		check.Set("status", g(session, "status"))
		if _, err := validatePendingAuthorizationStep.Run(e, check); err != nil {
			return nil, err
		}
		payment, err := paymentsteps.AuthorizePaymentSessionStep.Run(e, paymentsteps.AuthorizePaymentSessionStepInput{ID: sessionID})
		if err != nil {
			return nil, err
		}
		pf := dal.NewRecord()
		pf.Set("payment_collection_id", g(session, "payment_collection_id"))
		isList := false
		opc, err := commonsteps.UseQueryGraphStep.Config("get-order-payment-collection", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "order_payment_collection", Fields: []string{"order.id"}, Filters: pf}, IsList: &isList})
		if err != nil {
			return nil, err
		}
		orderID := g(rec(g(rec(opc.Data), "order")), "id")
		if payment != nil && orderID != nil {
			var tx []*dal.Record
			for _, c := range recs(g(payment, "captures")) {
				t := dal.NewRecord()
				t.Set("order_id", orderID)
				t.Set("amount", g(c, "amount"))
				t.Set("currency_code", g(payment, "currency_code"))
				t.Set("reference_id", g(c, "id"))
				t.Set("reference", "capture")
				tx = append(tx, t)
			}
			if _, err := ordersteps.AddOrderTransactionStep.Run(e, tx); err != nil {
				return nil, err
			}
		}
		return payment, nil
	})
