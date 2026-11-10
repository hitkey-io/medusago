package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type exchangeOps interface {
	Internal(className string) *utilsmodulessdk.InternalService
	CreateOrderExchanges(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	CreateOrderExchangeItems(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	DeleteOrderExchangeItems(ctx context.Context, ids []any) error
}

func exchangesOf(c composer.Container) exchangeOps {
	return c.Resolve(utilsmodulessdk.ORDER).(exchangeOps)
}

// CreateOrderExchangesStep mirrors createOrderExchangesStep.
var CreateOrderExchangesStep = composer.CreateStep("create-order-exchanges",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := exchangesOf(ctx.Container).CreateOrderExchanges(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(ids(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		list, _ := data.([]any)
		if list == nil {
			return nil
		}
		_, err := exchangesOf(ctx.Container).Internal("OrderExchange").Delete(ctx.Context, list)
		return err
	})

// CreateExchangeItemsFromActionsInput mirrors CreateOrderExchangeItemsFromActionsInput.
type CreateExchangeItemsFromActionsInput struct {
	Changes    []*dal.Record
	ExchangeID any
}

// CreateOrderExchangeItemsFromActionsStep mirrors createOrderExchangeItemsFromActionsStep.
var CreateOrderExchangeItemsFromActionsStep = composer.CreateStep("create-exchange-items-from-change-actions",
	func(ctx composer.StepExecutionContext, in CreateExchangeItemsFromActionsInput) (composer.StepResponse[[]*dal.Record], error) {
		items := make([]*dal.Record, 0, len(in.Changes))
		for _, c := range in.Changes {
			details, _ := get(c, "details").(*dal.Record)
			r := dal.NewRecord()
			r.Set("exchange_id", in.ExchangeID)
			r.Set("item_id", get(details, "reference_id"))
			r.Set("quantity", get(details, "quantity"))
			if v, ok := c.Get("internal_note"); ok {
				r.Set("note", v)
			}
			if m := get(details, "metadata"); m != nil {
				r.Set("metadata", m)
			} else {
				r.Set("metadata", dal.NewRecord())
			}
			items = append(items, r)
		}
		created, err := exchangesOf(ctx.Container).CreateOrderExchangeItems(ctx.Context, items)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(ids(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		list, _ := data.([]any)
		if list == nil {
			return nil
		}
		return exchangesOf(ctx.Container).DeleteOrderExchangeItems(ctx.Context, list)
	})

// UpdateOrderExchangesStep mirrors updateOrderExchangesStep.
var UpdateOrderExchangesStep = updateEntitiesStep("update-order-exchange", "OrderExchange", "OrderExchanges")

// DeleteExchangesStep mirrors deleteExchangesStep.
var DeleteExchangesStep = softDeleteStep("delete-exchanges", "OrderExchange")
