package subscribers

import (
	"context"
	"strconv"

	"github.com/hitkey-io/medusago/packages/core/framework/src/plugins"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	gcworkflows "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/gift-cards/workflows"
)

// createGiftCard mirrors create-gift-card.ts: on order.placed, a gift card
// per unit of each gift card line item (a product with is_giftcard), worth
// the item subtotal divided by its quantity.
var createGiftCard = plugins.Subscriber{
	Events:  []string{"order.placed"},
	Handler: createGiftCardHandler,
}

func gcNumber(v any) float64 {
	f, _ := strconv.ParseFloat(totals.Convert(v).Text('f'), 64)
	return f
}

func gcGet(r any, k string) any {
	rec, _ := r.(*dal.Record)
	if rec == nil {
		return nil
	}
	v, _ := rec.Get(k)
	return v
}

func createGiftCardHandler(ctx context.Context, c plugins.Resolver, ev eventbus.Event) error {
	orderID := gcGet(ev.Data, "id")
	q, _ := c.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)
	filters := dal.NewRecord()
	filters.Set("id", orderID)
	res, err := q.Graph(ctx, query.GraphInput{
		Entity:  "order",
		Filters: filters,
		Fields: []string{
			"id", "currency_code", "items.id", "items.subtotal", "items.quantity", "items.*",
			"items.product.is_giftcard", "total", "subtotal", "tax_total",
		},
	}, joiner.Options{})
	if err != nil {
		return err
	}
	var order any
	if len(res.Data) > 0 {
		order = res.Data[0]
	}
	items, _ := gcGet(order, "items").([]any)
	var giftCardItems []any
	for _, item := range items {
		if is, _ := gcGet(gcGet(item, "product"), "is_giftcard").(bool); is {
			giftCardItems = append(giftCardItems, item)
		}
	}
	if len(giftCardItems) == 0 {
		return nil
	}
	for _, item := range giftCardItems {
		quantity := gcNumber(gcGet(item, "quantity"))
		for i := 0; float64(i) < quantity; i++ {
			value := gcNumber(gcGet(item, "subtotal")) / quantity
			giftCard := dal.NewRecord()
			giftCard.Set("value", value)
			giftCard.Set("currency_code", gcGet(order, "currency_code"))
			giftCard.Set("line_item_id", gcGet(item, "id"))
			giftCard.Set("reference", "order")
			giftCard.Set("reference_id", gcGet(order, "id"))
			giftCard.Set("metadata", dal.NewRecord())
			if _, err := gcworkflows.CreateGiftCardsWorkflow.Run(ctx, c, []*dal.Record{giftCard}); err != nil {
				return err
			}
		}
	}
	return nil
}
