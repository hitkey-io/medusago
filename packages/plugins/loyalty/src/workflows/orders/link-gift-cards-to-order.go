// Package orders mirrors src/workflows/orders of @medusajs/loyalty-plugin.
package orders

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	commonworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/workflows"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/loyalty"
)

// CloneCartGiftCardsToOrderWorkflowInput mirrors CloneCartGiftCardsToOrderWorkflowInput.
type CloneCartGiftCardsToOrderWorkflowInput struct {
	OrderID string `json:"order_id"`
	CartID  string `json:"cart_id"`
}

// gcQueryGraph mirrors useQueryGraphStep({ ..., options: {
// throwIfKeyNotFound: true } }).config({ name }): the rows.
func gcQueryGraph(e *composer.Execution, name, entity string, fields []string, filters *dal.Record) ([]any, error) {
	res, err := commonsteps.UseQueryGraphStep.Config(name, composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
		GraphInput: query.GraphInput{Entity: entity, Fields: fields, Filters: filters},
		Options:    joiner.Options{ThrowIfKeyNotFound: true},
	})
	if err != nil {
		return nil, err
	}
	rows, _ := res.Data.([]any)
	return rows, nil
}

func gcRecord(kv ...any) *dal.Record {
	r := dal.NewRecord()
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i].(string), kv[i+1])
	}
	return r
}

func gcGet(r any, k string) any {
	rec, _ := r.(*dal.Record)
	if rec == nil {
		return nil
	}
	v, _ := rec.Get(k)
	return v
}

func gcList(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case []*dal.Record:
		out := make([]any, len(t))
		for i, r := range t {
			out[i] = r
		}
		return out
	}
	return nil
}

func gcFirst(rows []any) any {
	if len(rows) == 0 {
		return nil
	}
	return rows[0]
}

// CloneCartGiftCardsToOrderWorkflow mirrors cloneCartGiftCardsToOrderWorkflow
// ("clone-cart-gift-cards-to-order"): the gift cards of the cart are linked
// to the order created from it.
var CloneCartGiftCardsToOrderWorkflow = composer.CreateWorkflow("clone-cart-gift-cards-to-order",
	func(e *composer.Execution, input CloneCartGiftCardsToOrderWorkflowInput) (any, error) {
		orderRows, err := gcQueryGraph(e, "get-order-query", "order", []string{"id", "currency_code", "total", "gift_cards.code", "customer_id"}, gcRecord("id", input.OrderID))
		if err != nil {
			return nil, err
		}
		order := gcFirst(orderRows)
		cartRows, err := gcQueryGraph(e, "get-cart-query", "cart", []string{"id", "currency_code", "total", "gift_cards.id", "customer_id"}, gcRecord("id", input.CartID))
		if err != nil {
			return nil, err
		}
		cart := gcFirst(cartRows)
		giftCardIDs := []any{}
		for _, gc := range gcList(gcGet(cart, "gift_cards")) {
			giftCardIDs = append(giftCardIDs, gcGet(gc, "id"))
		}
		giftCards, err := gcQueryGraph(e, "get-gift-card-query", "gift_card", []string{"id", "code", "status", "customer_id", "currency_code"}, gcRecord("id", giftCardIDs))
		if err != nil {
			return nil, err
		}
		links := make([]*dal.Record, 0, len(giftCards))
		for _, gc := range giftCards {
			links = append(links, gcRecord(
				utilsmodulessdk.ORDER, gcRecord("order_id", gcGet(order, "id")),
				loyalty.ServiceName, gcRecord("gift_card_id", gcGet(gc, "id")),
			))
		}
		if _, err := commonworkflows.CreateLinksWorkflow.RunAsStep(e, links); err != nil {
			return nil, err
		}
		return nil, nil
	})
