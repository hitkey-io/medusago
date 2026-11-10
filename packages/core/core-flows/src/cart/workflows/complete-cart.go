package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	paymentsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment/steps"
	promotionsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/promotion/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CompleteCartWorkflowID mirrors completeCartWorkflowId.
const CompleteCartWorkflowID = "complete-cart"

func boolPtr(b bool) *bool { return &b }

// CompleteCartWorkflow mirrors completeCartWorkflow. The locking module is
// not ported (acquire/release lock steps are no-ops) and the workflow
// execution is not stored.
var CompleteCartWorkflow = composer.CreateWorkflowWithOptions(CompleteCartWorkflowID, composer.WorkflowOptions{Store: true, RetentionTime: 60 * 60 * 24 * 3},
	func(e *composer.Execution, cartID string) (*dal.Record, error) {
		var orderCart, cartData commonsteps.QueryGraphResult
		err := composer.Parallelize(
			func() error {
				f := dal.NewRecord()
				f.Set("cart_id", cartID)
				var err error
				orderCart, err = commonsteps.UseQueryGraphStep.Run(e, commonsteps.UseQueryGraphStepInput{
					GraphInput: query.GraphInput{Entity: "order_cart", Fields: []string{"cart_id", "order_id"}, Filters: f},
					IsList:     boolPtr(false),
				})
				return err
			},
			func() error {
				f := dal.NewRecord()
				f.Set("id", cartID)
				var err error
				cartData, err = commonsteps.UseQueryGraphStep.Config("cart-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
					GraphInput: query.GraphInput{Entity: "cart", Fields: CompleteCartFields, Filters: f},
					IsList:     boolPtr(false),
				})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		orderID := Get(asRecord(orderCart.Data), "order_id")
		cart := asRecord(cartData.Data)

		if _, err := steps.ValidateCartItemsStep.Run(e, cart); err != nil {
			return nil, err
		}
		paymentSessions, err := steps.ValidateCartPaymentsStep.Run(e, cart)
		if err != nil {
			return nil, err
		}
		var firstSessionID any
		if len(paymentSessions) > 0 {
			firstSessionID = Get(paymentSessions[0], "id")
		}
		if _, err := steps.CompensatePaymentIfNeededStep.Run(e, firstSessionID); err != nil {
			return nil, err
		}
		hookIn := dal.NewRecord()
		hookIn.Set("input", cartID)
		hookIn.Set("cart", cart)
		if err := composer.CreateHook(e, "validate", hookIn); err != nil {
			return nil, err
		}

		var order *dal.Record
		if !Truthy(orderID) {
			order, err = completeCartCreateOrder(e, cart, firstSessionID)
			if err != nil {
				return nil, err
			}
		}

		out := dal.NewRecord()
		if order != nil {
			out.Set("id", Get(order, "id"))
		} else {
			out.Set("id", orderID)
		}
		return out, nil
	})

func asRecord(v any) *dal.Record {
	r, _ := v.(*dal.Record)
	return r
}

func completeCartCreateOrder(e *composer.Execution, cart *dal.Record, paymentSessionID any) (*dal.Record, error) {
	var optionIDs []any
	for _, sm := range Records(Get(cart, "shipping_methods")) {
		optionIDs = append(optionIDs, Get(sm, "shipping_option_id"))
	}
	of := dal.NewRecord()
	of.Set("id", optionIDs)
	shippingOptions, err := commonsteps.UseQueryGraphStep.Config("shipping-options-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
		GraphInput: query.GraphInput{Entity: "shipping_option", Fields: []string{"id", "shipping_profile_id"}, Filters: of},
		Options:    joiner.Options{},
	})
	if err != nil {
		return nil, err
	}
	if _, err := steps.ValidateShippingStep.Run(e, steps.ValidateShippingStepInput{Cart: cart, ShippingOptions: Records(shippingOptions.Data)}); err != nil {
		return nil, err
	}

	var variantOrder []any
	variants := map[any]*dal.Record{}
	for _, item := range Records(Get(cart, "items")) {
		id := Get(item, "variant_id")
		if _, ok := variants[id]; !ok {
			variantOrder = append(variantOrder, id)
		}
		variants[id] = Rec(item, "variant")
	}
	var variantList []*dal.Record
	for _, id := range variantOrder {
		variantList = append(variantList, variants[id])
	}
	salesChannelID, _ := Get(cart, "sales_channel_id").(string)

	cartToOrder, err := prepareCartToOrder(cart)
	if err != nil {
		return nil, err
	}
	createdOrders, err := ordersteps.CreateOrdersStep.Run(e, []*dal.Record{cartToOrder})
	if err != nil {
		return nil, err
	}
	createdOrder := createdOrders[0]

	var reservationItems []*dal.Record
	for _, i := range Records(Get(createdOrder, "items")) {
		r := dal.NewRecord()
		r.Set("variant_id", Get(i, "variant_id"))
		r.Set("quantity", Get(i, "quantity"))
		r.Set("id", Get(i, "id"))
		reservationItems = append(reservationItems, r)
	}
	inventoryItems, err := PrepareConfirmInventoryInput(ConfirmVariantInventoryInput{SalesChannelID: salesChannelID, Variants: variantList, Items: reservationItems})
	if err != nil {
		return nil, err
	}

	updateCompletedAt := dal.NewRecord()
	updateCompletedAt.Set("id", Get(cart, "id"))
	updateCompletedAt.Set("completed_at", dal.Now())

	var usage []*dal.Record
	addUsage := func(adjustments []*dal.Record) {
		for _, a := range adjustments {
			u := dal.NewRecord()
			u.Set("amount", Get(a, "amount"))
			u.Set("code", Get(a, "code"))
			usage = append(usage, u)
		}
	}
	for _, item := range Records(Get(cart, "items")) {
		addUsage(Records(Get(item, "adjustments")))
	}
	for _, sm := range Records(Get(cart, "shipping_methods")) {
		addUsage(Records(Get(sm, "adjustments")))
	}
	registration := dal.NewRecord()
	registration.Set("customer_id", Coalesce(nilIfFalsy(Get(Rec(cart, "customer"), "id")), nil))
	registration.Set("customer_email", nilIfFalsy(Get(cart, "email")))

	link := func(a string, ak string, av any, b string, bk string, bv any) *dal.Record {
		l := dal.NewRecord()
		x := dal.NewRecord()
		x.Set(ak, av)
		y := dal.NewRecord()
		y.Set(bk, bv)
		l.Set(a, x)
		l.Set(b, y)
		return l
	}
	orderIDValue := Get(createdOrder, "id")
	links := []*dal.Record{link(utilsmodulessdk.ORDER, "order_id", orderIDValue, utilsmodulessdk.CART, "cart_id", Get(cart, "id"))}
	for _, p := range Records(Get(cart, "promotions")) {
		links = append(links, link(utilsmodulessdk.ORDER, "order_id", orderIDValue, utilsmodulessdk.PROMOTION, "promotion_id", Get(p, "id")))
	}
	if pc := Rec(cart, "payment_collection"); pc != nil && Get(pc, "id") != nil {
		links = append(links, link(utilsmodulessdk.ORDER, "order_id", orderIDValue, utilsmodulessdk.PAYMENT, "payment_collection_id", Get(pc, "id")))
	}

	var createdReservations []*dal.Record
	placed := dal.NewRecord()
	placed.Set("id", orderIDValue)
	err = composer.Parallelize(
		func() error { _, err := commonsteps.CreateRemoteLinkStep.Run(e, links); return err },
		func() error { _, err := UpdateCartsStep.Run(e, []*dal.Record{updateCompletedAt}); return err },
		func() error {
			var err error
			createdReservations, err = steps.ReserveInventoryStep.Run(e, inventoryItems)
			return err
		},
		func() error {
			_, err := promotionsteps.RegisterUsageStep.Run(e, promotionsteps.RegisterUsageStepInput{ComputedActions: usage, RegistrationContext: registration})
			return err
		},
		func() error {
			_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "order.placed", Data: placed})
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	var reservationEvents []any
	for _, r := range createdReservations {
		ev := dal.NewRecord()
		ev.Set("id", Get(r, "id"))
		ev.Set("order_id", orderIDValue)
		reservationEvents = append(reservationEvents, ev)
	}
	if reservationEvents == nil {
		reservationEvents = []any{}
	}
	if _, err := commonsteps.EmitEventStep.Config("emit-reservation-item-created", composer.StepOptions{}).Run(e, commonsteps.EmitEventStepInput{EventName: "reservation_item.created", Data: reservationEvents}); err != nil {
		return nil, err
	}
	hookIn := dal.NewRecord()
	hookIn.Set("input", Get(cart, "id"))
	if err := composer.CreateHook(e, "beforePaymentAuthorization", hookIn); err != nil {
		return nil, err
	}

	sessionID, _ := paymentSessionID.(string)
	payment, err := paymentsteps.AuthorizePaymentSessionStep.Run(e, paymentsteps.AuthorizePaymentSessionStepInput{ID: sessionID})
	if err != nil {
		return nil, err
	}
	var transactions []*dal.Record
	if payment != nil {
		for _, c := range Records(Get(payment, "captures")) {
			t := dal.NewRecord()
			t.Set("order_id", orderIDValue)
			t.Set("amount", Coalesce(Get(c, "raw_amount"), Get(c, "amount")))
			t.Set("currency_code", Get(payment, "currency_code"))
			t.Set("reference", "capture")
			t.Set("reference_id", Get(c, "id"))
			transactions = append(transactions, t)
		}
	}
	if _, err := ordersteps.AddOrderTransactionStep.Run(e, transactions); err != nil {
		return nil, err
	}
	created := dal.NewRecord()
	created.Set("order_id", orderIDValue)
	created.Set("cart_id", Get(cart, "id"))
	if err := composer.CreateHook(e, "orderCreated", created); err != nil {
		return nil, err
	}
	return createdOrder, nil
}

func nilIfFalsy(v any) any {
	if !Truthy(v) {
		return nil
	}
	return v
}

// prepareCartToOrder mirrors the cartToOrder transform.
func prepareCartToOrder(cart *dal.Record) (*dal.Record, error) {
	var allItems []any
	var itemAdjustments []*dal.Record
	for _, item := range Records(Get(cart, "items")) {
		isTaxInclusive, _ := Get(item, "is_tax_inclusive").(bool)
		taxLines := Records(Get(item, "tax_lines"))
		if taxLines == nil {
			taxLines = []*dal.Record{}
		}
		adjustments := Records(Get(item, "adjustments"))
		if adjustments == nil {
			adjustments = []*dal.Record{}
		}
		li, err := PrepareLineItemData(PrepareLineItemDataInput{
			Item: item, Variant: Rec(item, "variant"), CartID: Str(cart, "id"), UnitPrice: Get(item, "unit_price"),
			IsTaxInclusive: isTaxInclusive, TaxLines: taxLines, Adjustments: adjustments,
		})
		if err != nil {
			return nil, err
		}
		allItems = append(allItems, li)
		itemAdjustments = append(itemAdjustments, Records(Get(li, "adjustments"))...)
	}
	var shippingMethods []any
	var shippingAdjustments []*dal.Record
	for _, sm := range Records(Get(cart, "shipping_methods")) {
		m := dal.NewRecord()
		m.Set("name", Get(sm, "name"))
		m.Set("description", Get(sm, "description"))
		m.Set("amount", Coalesce(Get(sm, "raw_amount"), Get(sm, "amount")))
		m.Set("is_tax_inclusive", Get(sm, "is_tax_inclusive"))
		m.Set("shipping_option_id", Get(sm, "shipping_option_id"))
		m.Set("data", Get(sm, "data"))
		m.Set("metadata", Get(sm, "metadata"))
		taxLines := Records(Get(sm, "tax_lines"))
		if taxLines == nil {
			taxLines = []*dal.Record{}
		}
		adjustments := Records(Get(sm, "adjustments"))
		if adjustments == nil {
			adjustments = []*dal.Record{}
		}
		m.Set("tax_lines", PrepareTaxLinesData(taxLines))
		m.Set("adjustments", PrepareAdjustmentsData(adjustments))
		shippingMethods = append(shippingMethods, m)
		shippingAdjustments = append(shippingAdjustments, Records(Get(m, "adjustments"))...)
	}
	var creditLines []any
	for _, cl := range Records(Get(cart, "credit_lines")) {
		c := dal.NewRecord()
		c.Set("amount", Get(cl, "amount"))
		c.Set("raw_amount", Get(cl, "raw_amount"))
		c.Set("reference", Get(cl, "reference"))
		c.Set("reference_id", Get(cl, "reference_id"))
		c.Set("metadata", Get(cl, "metadata"))
		creditLines = append(creditLines, c)
	}
	var promoCodes []any
	for _, a := range append(itemAdjustments, shippingAdjustments...) {
		if Truthy(Get(a, "code")) {
			promoCodes = append(promoCodes, Get(a, "code"))
		}
	}
	address := func(k string) any {
		a := Rec(cart, k)
		if a == nil {
			return nil
		}
		c := Copy(a)
		c.Delete("id")
		return c
	}
	orEmpty := func(l []any) []any {
		if l == nil {
			return []any{}
		}
		return l
	}
	o := dal.NewRecord()
	o.Set("region_id", Get(Rec(cart, "region"), "id"))
	o.Set("customer_id", Get(Rec(cart, "customer"), "id"))
	o.Set("sales_channel_id", Get(cart, "sales_channel_id"))
	o.Set("status", "pending")
	o.Set("email", Get(cart, "email"))
	o.Set("currency_code", Get(cart, "currency_code"))
	o.Set("locale", Get(cart, "locale"))
	o.Set("shipping_address", address("shipping_address"))
	o.Set("billing_address", address("billing_address"))
	o.Set("no_notification", false)
	o.Set("items", orEmpty(allItems))
	o.Set("shipping_methods", orEmpty(shippingMethods))
	o.Set("metadata", Get(cart, "metadata"))
	o.Set("promo_codes", orEmpty(promoCodes))
	o.Set("credit_lines", orEmpty(creditLines))
	return o, nil
}
