// Package steps mirrors @medusajs/core-flows/src/order/steps.
package steps

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// OrderService is the part of the order module the steps use.
type OrderService interface {
	Internal(className string) *utilsmodulessdk.InternalService
	ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error)
	CreateOrders(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	AddOrderTransactions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
}

// OrderOf resolves the order module service.
func OrderOf(c composer.Container) OrderService {
	s, _ := c.Resolve(utilsmodulessdk.ORDER).(OrderService)
	return s
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func ids(list []*dal.Record) []any {
	out := make([]any, 0, len(list))
	for _, r := range list {
		out = append(out, get(r, "id"))
	}
	return out
}

// CreateOrdersStep mirrors createOrdersStep.
var CreateOrdersStep = composer.CreateStep("create-orders",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := OrderOf(ctx.Container).CreateOrders(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(ids(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		createdIDs, _ := data.([]any)
		if len(createdIDs) == 0 {
			return nil
		}
		_, err := OrderOf(ctx.Container).Internal("Order").Delete(ctx.Context, createdIDs)
		return err
	})

// AddOrderTransactionStep mirrors addOrderTransactionStep (array input).
var AddOrderTransactionStep = composer.CreateStep("add-order-transaction",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(data) == 0 {
			return composer.NewStepResponse[[]*dal.Record](nil), nil
		}
		svc := OrderOf(ctx.Container)
		var or []any
		for _, t := range data {
			q := dal.NewRecord()
			q.Set("order_id", get(t, "order_id"))
			q.Set("reference", get(t, "reference"))
			q.Set("reference_id", get(t, "reference_id"))
			or = append(or, q)
		}
		f := dal.NewRecord()
		f.Set("$or", or)
		existing, err := svc.ListBySuffix(ctx.Context, "OrderTransactions", f, utilsmodulessdk.FindConfig{Select: []string{"order_id", "reference", "reference_id"}})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		key := func(r *dal.Record) string {
			return fmt.Sprintf("%v-%v-%v", get(r, "order_id"), get(r, "reference"), get(r, "reference_id"))
		}
		seen := map[string]bool{}
		for _, e := range existing {
			seen[key(e)] = true
		}
		var selected []*dal.Record
		for _, t := range data {
			if !seen[key(t)] {
				selected = append(selected, t)
			}
		}
		if len(selected) == 0 {
			return composer.NewStepResponse[[]*dal.Record](nil), nil
		}
		created, err := svc.AddOrderTransactions(ctx.Context, selected)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(ids(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		createdIDs, _ := data.([]any)
		if len(createdIDs) == 0 {
			return nil
		}
		_, err := OrderOf(ctx.Container).Internal("OrderTransaction").Delete(ctx.Context, createdIDs)
		return err
	})
