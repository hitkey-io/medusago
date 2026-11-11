package workflows

import (
	"fmt"
	"strings"

	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	fulfillmentsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/steps"
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

// remoteOne mirrors useRemoteQueryStep({ list: false, throw_if_key_not_found }).
func remoteOne(e *composer.Execution, name, entry string, variables *dal.Record, fields []string, throwIfNotFound bool) (*dal.Record, error) {
	list := false
	step := commonsteps.UseRemoteQueryStep
	if name != "" {
		step = step.Config(name, composer.StepOptions{})
	}
	res, err := step.Run(e, commonsteps.RemoteStepInput{EntryPoint: entry, Fields: fields, Variables: variables, List: &list, ThrowIfKeyNotFound: throwIfNotFound})
	if err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, nil
	}
	return rec(res.Data[0]), nil
}

func idVars(id any) *dal.Record {
	v := dal.NewRecord()
	v.Set("id", id)
	return v
}

// activeOrderChangeFor queries the pending/requested order change of a
// related entity (return/claim/exchange).
func activeOrderChangeFor(e *composer.Execution, orderID any, key string, id any, fields []string) (*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("order_id", orderID)
	f.Set(key, id)
	f.Set("status", []any{"pending", "requested"})
	v := dal.NewRecord()
	v.Set("filters", f)
	return remoteOne(e, "order-change-query", "order_change", v, fields, false)
}

func throwIfIsCancelled(obj *dal.Record, typ string) error {
	if truthy(g(obj, "canceled_at")) {
		return invalidData(fmt.Sprintf("%s with id %v has been canceled.", typ, g(obj, "id")))
	}
	return nil
}

func ThrowIfOrderChangeIsNotActive(orderChange *dal.Record) error {
	if orderChange == nil || orderChange.Len() == 0 {
		return invalidData("An active Order Change is required to proceed")
	}
	if truthy(g(orderChange, "canceled_at")) || truthy(g(orderChange, "confirmed_at")) || truthy(g(orderChange, "declined_at")) {
		return invalidData(fmt.Sprintf("Order change %v is not active to be modified", g(orderChange, "id")))
	}
	return nil
}

func arrayDifference(main, exclude []any) []string {
	ex := map[string]bool{}
	for _, v := range exclude {
		ex[fmt.Sprint(v)] = true
	}
	var out []string
	for _, v := range main {
		if !ex[fmt.Sprint(v)] {
			out = append(out, fmt.Sprint(v))
		}
	}
	return out
}

func throwIfItemsDoesNotExistsInReturn(orderReturn *dal.Record, inputItems []*dal.Record) error {
	var existing, input []any
	for _, i := range recs(g(orderReturn, "items")) {
		existing = append(existing, g(i, "item_id"))
	}
	for _, i := range inputItems {
		input = append(input, g(i, "id"))
	}
	if diff := arrayDifference(input, existing); len(diff) > 0 {
		return invalidData(fmt.Sprintf("Items with ids %s does not exist in Return with id %v.", strings.Join(diff, ", "), g(orderReturn, "id")))
	}
	return nil
}

// validateReturnReasons mirrors validateReturnReasons.
func validateReturnReasons(ctx composer.StepExecutionContext, orderID any, inputItems []*dal.Record) error {
	var reasonIDs, all []any
	for _, i := range inputItems {
		r := g(i, "reason_id")
		all = append(all, r)
		if truthy(r) {
			reasonIDs = append(reasonIDs, r)
		}
	}
	if len(reasonIDs) == 0 {
		return nil
	}
	v := dal.NewRecord()
	v.Set("id", reasonIDs)
	v.Set("limit", nil)
	rows, err := commonsteps.RemoteQuery(ctx, "return_reasons", v, []string{"id", "parent_return_reason_id", "parent_return_reason", "return_reason_children.id"})
	if err != nil {
		return err
	}
	var found []any
	var nested []string
	for _, row := range rows {
		r := rec(row)
		found = append(found, g(r, "id"))
		if len(recs(g(r, "return_reason_children"))) > 0 {
			nested = append(nested, fmt.Sprint(g(r, "id")))
		}
	}
	if missing := arrayDifference(reasonIDs, found); len(missing) > 0 {
		return invalidData(fmt.Sprintf("Return reason with id %s does not exists.", strings.Join(missing, ", ")))
	}
	if len(nested) > 0 {
		return invalidData(fmt.Sprintf("Cannot apply return reason with id %s to order with id %v. Return reason has nested reasons.", strings.Join(nested, ", "), orderID))
	}
	_ = all
	return nil
}

// FetchShippingOptionInput mirrors FetchShippingOptionForOrderWorkflowInput.
type FetchShippingOptionInput struct {
	ShippingOptionID any
	CurrencyCode     any
	OrderID          any
	Context          *dal.Record
}

// calculateOrderShippingPrice mirrors the calculated branch shared by
// fetchShippingOptionForOrderWorkflow and maybeRefreshShippingMethodsWorkflow:
// the provider prices the option for the order (and the context given).
//
// maybeRefreshShippingMethodsWorkflow reads the order with
// `transform(orderQuery, (data) => data[0])` on the query result object, so
// its order is undefined and only the given context reaches the provider
// (spreadOrder false).
func calculateOrderShippingPrice(e *composer.Execution, option *dal.Record, orderID any, context_ *dal.Record, orderQueryName string, spreadOrder bool) ([]*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("id", orderID)
	isList := false
	res, err := commonsteps.UseQueryGraphStep.Config(orderQueryName, composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
		GraphInput: query.GraphInput{Entity: "order", Fields: []string{"id", "shipping_address", "items.*", "items.variant.*"}, Filters: f},
		IsList:     &isList,
		Options:    joiner.Options{ThrowIfKeyNotFound: true},
	})
	if err != nil {
		return nil, err
	}
	order := rec(res.Data)
	if order == nil {
		if list := recs(res.Data); len(list) > 0 {
			order = list[0]
		}
	}
	if !spreadOrder {
		order = nil
	}
	c := dal.NewRecord()
	for _, src := range []*dal.Record{order, context_} {
		if src == nil {
			continue
		}
		for _, k := range src.Keys() {
			v, _ := src.Get(k)
			c.Set(k, v)
		}
	}
	if loc := g(rec(g(rec(g(option, "service_zone")), "fulfillment_set")), "location"); loc != nil {
		c.Set("from_location", loc)
	}
	d := dal.NewRecord()
	d.Set("id", g(option, "id"))
	d.Set("optionData", g(option, "data"))
	d.Set("context", c)
	d.Set("provider_id", g(option, "provider_id"))
	return fulfillmentsteps.CalculateShippingOptionsPricesStep.Run(e, []*dal.Record{d})
}

// FetchShippingOptionForOrderWorkflow mirrors fetchShippingOptionForOrderWorkflow.
var FetchShippingOptionForOrderWorkflow = composer.CreateWorkflow("fetch-shipping-option",
	func(e *composer.Execution, in FetchShippingOptionInput) (*dal.Record, error) {
		f := dal.NewRecord()
		f.Set("id", in.ShippingOptionID)
		isList := false
		res, err := commonsteps.UseQueryGraphStep.Config("shipping-option-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "shipping_option", Fields: []string{"id", "price_type"}, Filters: f},
			IsList:     &isList,
		})
		if err != nil {
			return nil, err
		}
		initial := rec(res.Data)
		hookIn := dal.NewRecord()
		hookIn.Set("input", in)
		if err := composer.CreateHook(e, "setCalculatedShippingPricingContext", hookIn); err != nil {
			return nil, err
		}
		if g(initial, "price_type") == "calculated" {
			of := dal.NewRecord()
			of.Set("id", in.ShippingOptionID)
			ores, err := commonsteps.UseQueryGraphStep.Config("calculated-option", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
				GraphInput: query.GraphInput{Entity: "shipping_option", Fields: shippingOptionCommonFields, Filters: of},
				IsList:     &isList,
			})
			if err != nil {
				return nil, err
			}
			option := rec(ores.Data)
			prices, err := calculateOrderShippingPrice(e, option, in.OrderID, in.Context, "order-query", true)
			if err != nil {
				return nil, err
			}
			out := dal.NewRecord()
			out.Set("id", g(option, "id"))
			out.Set("name", g(option, "name"))
			if len(prices) > 0 {
				out.Set("calculated_price", prices[0])
			}
			return out, nil
		}
		if err := composer.CreateHook(e, "setPricingContext", in); err != nil {
			return nil, err
		}
		pc := dal.NewRecord()
		pc.Set("currency_code", in.CurrencyCode)
		cp := dal.NewRecord()
		cp.Set("context", pc)
		v := dal.NewRecord()
		v.Set("id", in.ShippingOptionID)
		v.Set("calculated_price", cp)
		return remoteOne(e, "flat-reate-option", "shipping_option", v, []string{"id", "name", "calculated_price.calculated_amount", "calculated_price.is_calculated_price_tax_inclusive"}, false)
	})

var shippingOptionCommonFields = []string{"id", "name", "price_type", "service_zone_id", "service_zone.fulfillment_set_id", "service_zone.fulfillment_set.type",
	"service_zone.fulfillment_set.location.*", "service_zone.fulfillment_set.location.address.*", "shipping_profile_id", "provider_id", "data",
	"type.id", "type.label", "type.description", "type.code", "provider.id", "provider.is_enabled", "rules.attribute", "rules.value", "rules.operator"}

// MaybeRefreshShippingMethodsWorkflow mirrors maybeRefreshShippingMethodsWorkflow:
// the price of a calculated option is computed again.
var MaybeRefreshShippingMethodsWorkflow = composer.CreateWorkflow("maybe-refresh-shipping-methods",
	func(e *composer.Execution, in *dal.Record) (any, error) {
		f := dal.NewRecord()
		f.Set("id", g(in, "shipping_method_id"))
		methods, err := queryList(e, "fetch-shipping-method", "order_shipping_method", f, []string{"id", "shipping_option_id"})
		if err != nil {
			return nil, err
		}
		var optionID any
		if len(methods) > 0 {
			optionID = g(methods[0], "shipping_option_id")
		}
		of := dal.NewRecord()
		of.Set("id", optionID)
		// The shipping option, its type, provider and the resolved location are all
		// computed from the response. These are what computation cannot see: the
		// service zone, fulfillment set and rules are selected without their ids,
		// and the fulfillment set resolves its location through a link row.
		options, err := queryListWithOptions(e, "calculated-option", "shipping_option", of, shippingOptionCommonFields, joiner.Options{Cache: caching.WithTags(true,
			"ServiceZone:list:*",
			"FulfillmentSet:list:*",
			"ShippingOptionRule:list:*",
			"LinkLocationFulfillmentSet:list:*",
		)})
		if err != nil {
			return nil, err
		}
		hookIn := dal.NewRecord()
		hookIn.Set("input", in)
		if err := composer.CreateHook(e, "setCalculatedShippingPricingContext", hookIn); err != nil {
			return nil, err
		}
		if len(options) > 0 && g(options[0], "price_type") == "calculated" {
			prices, err := calculateOrderShippingPrice(e, options[0], g(in, "order_id"), rec(g(in, "context")), "order-query", false)
			if err != nil {
				return nil, err
			}
			var amount any
			if len(prices) > 0 {
				amount = g(prices[0], "calculated_amount")
			}
			action := dal.NewRecord()
			action.Set("id", g(in, "action_id"))
			action.Set("amount", amount)
			method := dal.NewRecord()
			method.Set("id", g(in, "shipping_method_id"))
			method.Set("amount", amount)
			method.Set("is_custom_amount", false)
			if err := composer.Parallelize(
				func() error {
					_, err := ordersteps.UpdateOrderChangeActionsStep.Run(e, []*dal.Record{action})
					return err
				},
				func() error {
					_, err := ordersteps.UpdateOrderShippingMethodsStep.Run(e, []*dal.Record{method})
					return err
				},
			); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})

// RefreshReturnShippingWorkflow mirrors refreshReturnShippingWorkflow.
var RefreshReturnShippingWorkflow = composer.CreateWorkflow("refresh-return-shipping",
	func(e *composer.Execution, in *dal.Record) (any, error) {
		f := dal.NewRecord()
		f.Set("order_id", g(in, "order_id"))
		f.Set("return_id", g(in, "return_id"))
		f.Set("status", []any{"pending", "requested"})
		changes, err := queryList(e, "order-change-query", "order_change", f, []string{"id", "status", "order_id", "return_id", "actions.*"})
		if err != nil {
			return nil, err
		}
		var orderChange *dal.Record
		if len(changes) > 0 {
			orderChange = changes[0]
		}
		var shippingAction *dal.Record
		items := []any{}
		for _, a := range recs(g(orderChange, "actions")) {
			if shippingAction == nil && g(a, "action") == "SHIPPING_ADD" {
				shippingAction = a
			}
			if g(a, "action") == "RETURN_ITEM" {
				d := rec(g(a, "details"))
				it := dal.NewRecord()
				it.Set("id", g(d, "reference_id"))
				it.Set("quantity", g(d, "quantity"))
				items = append(items, it)
			}
		}
		if shippingAction == nil {
			return nil, nil
		}
		c := dal.NewRecord()
		c.Set("return_id", g(in, "return_id"))
		c.Set("return_items", items)
		args := dal.NewRecord()
		args.Set("shipping_method_id", g(shippingAction, "reference_id"))
		args.Set("order_id", g(orderChange, "order_id"))
		args.Set("action_id", g(shippingAction, "id"))
		args.Set("context", c)
		return MaybeRefreshShippingMethodsWorkflow.RunAsStep(e, args)
	})

// prepareShippingMethod mirrors PrepareShippingMethod(relatedEntityField).
func PrepareShippingMethod(relatedEntityField string, option, orderChange, relatedEntity, input *dal.Record, customPrice any) *dal.Record {
	cp := rec(g(option, "calculated_price"))
	obj := dal.NewRecord()
	obj.Set("shipping_option_id", g(option, "id"))
	isCustom := customPrice != nil
	if isCustom {
		obj.Set("amount", customPrice)
	} else {
		obj.Set("amount", g(cp, "calculated_amount"))
	}
	obj.Set("is_custom_amount", isCustom)
	obj.Set("is_tax_inclusive", truthy(g(cp, "is_calculated_price_tax_inclusive")))
	if d := g(option, "data"); d != nil {
		obj.Set("data", d)
	} else {
		obj.Set("data", dal.NewRecord())
	}
	obj.Set("name", g(option, "name"))
	obj.Set("version", g(orderChange, "version"))
	obj.Set("order_id", g(relatedEntity, "order_id"))
	if relatedEntityField != "" {
		if v, ok := input.Get("return_id"); ok {
			obj.Set("return_id", v)
		}
		obj.Set(relatedEntityField, g(relatedEntity, "id"))
		if relatedEntityField == "return_id" {
			obj.Set("claim_id", g(relatedEntity, "claim_id"))
			obj.Set("exchange_id", g(relatedEntity, "exchange_id"))
		}
	}
	return obj
}

var getOrderPendingAmountStep = composer.CreateStep("get-order-pending-amount",
	func(ctx composer.StepExecutionContext, in struct {
		Order  *dal.Record
		Amount any
	}) (composer.StepResponse[any], error) {
		amount := in.Amount
		if amount == nil {
			amount = 0
		}
		summary := rec(g(in.Order, "summary"))
		pending := g(summary, "raw_pending_difference")
		if pending == nil {
			pending = g(summary, "pending_difference")
		}
		if totals.Gt(amount, 0) && totals.Gt(amount, pending) {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, fmt.Sprintf("Amount cannot be greater than %v", totals.Convert(pending).String()))
		}
		return composer.NewStepResponse(pending), nil
	}, nil)

var validatePaymentCollectionCancellationStep = composer.CreateStep("validate-payment-collection-cancellation",
	func(ctx composer.StepExecutionContext, pc *dal.Record) (composer.StepResponse[any], error) {
		switch g(pc, "status") {
		case "completed":
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Cannot cancel a completed payment collection")
		case "canceled":
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Payment collection is already canceled")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// CancelPaymentCollectionWorkflow mirrors cancelPaymentCollectionWorkflow.
var CancelPaymentCollectionWorkflow = composer.CreateWorkflow("cancel-payment-collection",
	func(e *composer.Execution, paymentCollectionID any) (any, error) {
		f := dal.NewRecord()
		f.Set("id", paymentCollectionID)
		list, err := queryList(e, "get-payment-collection", "payment_collection", f, []string{"id", "status", "payments.id", "payments.captured_at", "captured_amount"})
		if err != nil {
			return nil, err
		}
		var pc *dal.Record
		if len(list) > 0 {
			pc = list[0]
		}
		if _, err := validatePaymentCollectionCancellationStep.Run(e, pc); err != nil {
			return nil, err
		}
		var authorized []any
		for _, p := range recs(g(pc, "payments")) {
			if !truthy(g(p, "captured_at")) {
				authorized = append(authorized, g(p, "id"))
			}
		}
		if authorized == nil {
			authorized = []any{}
		}
		status := "canceled"
		if ca := g(pc, "captured_amount"); ca != nil && totals.Gt(ca, 0) {
			status = "partially_captured"
		}
		sel := dal.NewRecord()
		sel.Set("id", g(pc, "id"))
		upd := dal.NewRecord()
		upd.Set("status", status)
		u := dal.NewRecord()
		u.Set("selector", sel)
		u.Set("update", upd)
		if _, err := cartworkflows.UpdatePaymentCollectionStep.Run(e, u); err != nil {
			return nil, err
		}
		if _, err := paymentsteps.CancelPaymentStep.Run(e, authorized); err != nil {
			return nil, err
		}
		return nil, nil
	})

// CreateOrUpdateOrderPaymentCollectionWorkflow mirrors
// createOrUpdateOrderPaymentCollectionWorkflow.
var CreateOrUpdateOrderPaymentCollectionWorkflow = composer.CreateWorkflow("create-or-update-order-payment-collection",
	func(e *composer.Execution, in *dal.Record) (any, error) {
		f := dal.NewRecord()
		f.Set("id", g(in, "order_id"))
		isList := false
		res, err := commonsteps.UseQueryGraphStep.Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "order", Fields: []string{"id", "summary", "total", "currency_code", "region_id"}, Filters: f},
			Options:    joiner.Options{ThrowIfKeyNotFound: true},
			IsList:     &isList,
		})
		if err != nil {
			return nil, err
		}
		order := rec(res.Data)
		of := dal.NewRecord()
		of.Set("order_id", g(order, "id"))
		opcs, err := queryList(e, "order-payment-collection-query", "order_payment_collection", of, []string{"payment_collection_id"})
		if err != nil {
			return nil, err
		}
		pcIDs := []any{}
		for _, o := range opcs {
			pcIDs = append(pcIDs, g(o, "payment_collection_id"))
		}
		pf := dal.NewRecord()
		pf.Set("id", pcIDs)
		pf.Set("status", []any{"not_paid", "awaiting", "authorized", "partially_authorized"})
		pres, err := commonsteps.UseQueryGraphStep.Config("payment-collection-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "payment_collection", Fields: []string{"id", "status"}, Filters: pf},
			IsList:     &isList,
		})
		if err != nil {
			return nil, err
		}
		existing := rec(pres.Data)
		shouldRecreate := g(existing, "status") == "authorized" || g(existing, "status") == "partially_authorized"
		pending, err := getOrderPendingAmountStep.Run(e, struct {
			Order  *dal.Record
			Amount any
		}{order, g(in, "amount")})
		if err != nil {
			return nil, err
		}
		hasExisting := truthy(g(existing, "id"))
		if hasExisting && !shouldRecreate && totals.Gte(pending, 0) {
			sel := dal.NewRecord()
			sel.Set("id", g(existing, "id"))
			upd := dal.NewRecord()
			upd.Set("amount", pending)
			u := dal.NewRecord()
			u.Set("selector", sel)
			u.Set("update", upd)
			if _, err := cartworkflows.UpdatePaymentCollectionStep.Run(e, u); err != nil {
				return nil, err
			}
		}
		if (!hasExisting || shouldRecreate) && totals.Gt(pending, 0) {
			ci := dal.NewRecord()
			ci.Set("order_id", g(order, "id"))
			ci.Set("amount", pending)
			if _, err := CreateOrderPaymentCollectionWorkflow.RunAsStep(e, ci); err != nil {
				return nil, err
			}
		}
		if hasExisting && shouldRecreate && totals.Gt(pending, 0) {
			if _, err := CancelPaymentCollectionWorkflow.RunAsStep(e, g(existing, "id")); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})

// TranslateShippingOption mirrors getTranslatedShippingOptionsStep({
// shippingOptions: [shippingOption], locale }) of the shipping method
// workflows: the option translated in the order locale.
func TranslateShippingOption(e *composer.Execution, option *dal.Record, locale any) (*dal.Record, error) {
	if option == nil {
		_, err := commonsteps.GetTranslatedShippingOptionsStep.Run(e, commonsteps.GetTranslatedShippingOptionsStepInput{ShippingOptions: []*dal.Record{}, Locale: str(locale)})
		return nil, err
	}
	out, err := commonsteps.GetTranslatedShippingOptionsStep.Run(e, commonsteps.GetTranslatedShippingOptionsStepInput{ShippingOptions: []*dal.Record{option}, Locale: str(locale)})
	if err != nil || len(out) == 0 {
		return nil, err
	}
	return out[0], nil
}
