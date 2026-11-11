package workflows

import (
	"fmt"

	cartsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

var updateOrderValidationStep = composer.CreateStep("update-order-validation",
	func(ctx composer.StepExecutionContext, in struct{ Order, Input *dal.Record }) (composer.StepResponse[any], error) {
		if err := throwIfOrderIsCancelled(in.Order); err != nil {
			return composer.StepResponse[any]{}, err
		}
		for _, k := range []string{"shipping_address", "billing_address"} {
			if cc := g(rec(g(in.Input, k)), "country_code"); truthy(cc) && g(rec(g(in.Order, k)), "country_code") != cc {
				return composer.StepResponse[any]{}, invalidData("Country code cannot be changed")
			}
		}
		if email, _ := g(in.Input, "email").(string); email != "" {
			if _, err := cartsteps.ValidateEmail(email); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// UpdateOrderWorkflow mirrors updateOrderWorkflow.
var UpdateOrderWorkflow = composer.CreateWorkflow("update-order-workflow",
	func(e *composer.Execution, input *dal.Record) (any, error) {
		f := dal.NewRecord()
		f.Set("id", g(input, "id"))
		res, err := commonsteps.UseQueryGraphStep.Config("order-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "order", Fields: []string{"id", "status", "email", "locale", "shipping_address.*", "billing_address.*", "metadata",
				"shipping_methods.id", "shipping_methods.name", "shipping_methods.shipping_option_id"}, Filters: f},
			Options: joiner.Options{ThrowIfKeyNotFound: true},
		})
		if err != nil {
			return nil, err
		}
		var order *dal.Record
		if list := recs(res.Data); len(list) > 0 {
			order = list[0]
		}
		if _, err := updateOrderValidationStep.Run(e, struct{ Order, Input *dal.Record }{order, input}); err != nil {
			return nil, err
		}
		var customer *dal.Record
		if !truthy(g(order, "email")) && truthy(g(input, "email")) {
			cd, err := cartsteps.FindOrCreateCustomerStep.Run(e, cartsteps.FindOrCreateCustomerInput{Email: g(input, "email")})
			if err != nil {
				return nil, err
			}
			customer = cd.Customer
		}
		update := dal.NewRecord()
		for _, k := range input.Keys() {
			v, _ := input.Get(k)
			update.Set(k, v)
		}
		for _, k := range []string{"shipping_address", "billing_address"} {
			if in := rec(g(input, k)); in != nil {
				addr := dal.NewRecord()
				if cur := rec(g(order, k)); cur != nil {
					for _, kk := range cur.Keys() {
						v, _ := cur.Get(kk)
						addr.Set(kk, v)
					}
				}
				for _, kk := range in.Keys() {
					v, _ := in.Get(kk)
					addr.Set(kk, v)
				}
				addr.Delete("id")
				update.Set(k, addr)
			}
		}
		if customer != nil {
			update.Set("customer_id", g(customer, "id"))
		}
		sel := dal.NewRecord()
		sel.Set("id", g(input, "id"))
		updated, err := ordersteps.UpdateOrdersStep.Run(e, ordersteps.UpdateOrdersStepInput{Selector: sel, Update: update})
		if err != nil {
			return nil, err
		}
		var updatedOrder *dal.Record
		if len(updated) > 0 {
			updatedOrder = updated[0]
		}
		var changes []*dal.Record
		change := func(typ string, old, new any) {
			c := dal.NewRecord()
			c.Set("change_type", "update_order")
			c.Set("order_id", g(input, "id"))
			c.Set("created_by", g(input, "user_id"))
			c.Set("confirmed_by", g(input, "user_id"))
			d := dal.NewRecord()
			d.Set("type", typ)
			d.Set("old", old)
			d.Set("new", new)
			c.Set("details", d)
			changes = append(changes, c)
		}
		if g(input, "shipping_address") != nil {
			change("shipping_address", g(order, "shipping_address"), g(updatedOrder, "shipping_address"))
		}
		if g(input, "billing_address") != nil {
			change("billing_address", g(order, "billing_address"), g(updatedOrder, "billing_address"))
		}
		if truthy(g(input, "email")) {
			change("email", g(order, "email"), g(input, "email"))
		}
		if input.Has("metadata") {
			change("metadata", g(order, "metadata"), g(input, "metadata"))
		}
		if l := g(input, "locale"); truthy(l) && l != g(order, "locale") {
			change("locale", g(order, "locale"), l)
		}
		if _, err := ordersteps.RegisterOrderChangesStep.Run(e, changes); err != nil {
			return nil, err
		}
		if l := g(input, "locale"); truthy(l) && l != g(order, "locale") {
			if err := RetranslateOrder(e, fmt.Sprint(g(input, "id")), fmt.Sprint(l), recs(g(order, "shipping_methods")), false); err != nil {
				return nil, err
			}
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "order.updated", Data: idEvent(g(input, "id"))}); err != nil {
			return nil, err
		}
		return nil, nil
	})

// RetranslateOrder runs the locale-changed steps of updateOrderWorkflow
// (items, shipping methods, tax lines) and of updateDraftOrderWorkflow
// (draftOrder: shipping methods, tax lines, items) in parallel.
func RetranslateOrder(e *composer.Execution, orderID, locale string, shippingMethods []*dal.Record, draftOrder bool) error {
	if shippingMethods == nil {
		shippingMethods = []*dal.Record{}
	}
	items := func() error {
		_, err := ordersteps.UpdateOrderItemsTranslationsStep.Run(e, ordersteps.UpdateOrderItemsTranslationsStepInput{OrderID: orderID, Locale: locale})
		return err
	}
	shipping := func() error {
		_, err := ordersteps.UpdateOrderShippingMethodsTranslationsStep.Run(e, ordersteps.UpdateOrderShippingMethodsTranslationsStepInput{ShippingMethods: shippingMethods, Locale: locale})
		return err
	}
	taxLines := func() error {
		_, err := ordersteps.UpdateOrderTaxLinesTranslationsStep.Run(e, ordersteps.UpdateOrderTaxLinesTranslationsStepInput{OrderID: orderID, Locale: locale})
		return err
	}
	if draftOrder {
		return composer.Parallelize(shipping, taxLines, items)
	}
	return composer.Parallelize(items, shipping, taxLines)
}
