package steps

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type changeOps interface {
	Internal(className string) *utilsmodulessdk.InternalService
	CreateOrderChange(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	AddOrderAction(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
}

type confirmer interface {
	ConfirmOrderChangeRecords(ctx context.Context, data []any) error
}

func changesOf(c composer.Container) changeOps { return c.Resolve(utilsmodulessdk.ORDER).(changeOps) }

// CreateOrderChangeStep mirrors createOrderChangeStep.
var CreateOrderChangeStep = composer.CreateStep("create-order-change",
	func(ctx composer.StepExecutionContext, data *dal.Record) (composer.StepResponse[*dal.Record], error) {
		created, err := changesOf(ctx.Container).CreateOrderChange(ctx.Context, []*dal.Record{data})
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(created[0], get(created[0], "id")), nil
	},
	func(ctx composer.StepExecutionContext, id any) error {
		if id == nil {
			return nil
		}
		_, err := changesOf(ctx.Container).Internal("OrderChange").Delete(ctx.Context, []any{id})
		return err
	})

// AddOrderActionsStep mirrors createEntitiesStep({ invokeMethod: "addOrderAction" }).
var AddOrderActionsStep = composer.CreateStep("create-entities-step",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := changesOf(ctx.Container).AddOrderAction(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(ids(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		list, _ := data.([]any)
		if len(list) == 0 {
			return nil
		}
		_, err := changesOf(ctx.Container).Internal("OrderChangeAction").Delete(ctx.Context, list)
		return err
	})

// ConfirmOrderChangesInput mirrors ConfirmOrderChangesInput.
type ConfirmOrderChangesInput struct {
	OrderID     any
	Changes     []*dal.Record
	ConfirmedBy any
}

// confirmOrderChangesInvoke is the invoke of confirmOrderChanges; withResult
// makes the confirmOrderChange result ({ items, shipping_methods,
// credit_lines }) the step output.
func confirmOrderChangesInvoke(withResult bool) func(composer.StepExecutionContext, ConfirmOrderChangesInput) (composer.StepResponse[any], error) {
	return func(ctx composer.StepExecutionContext, in ConfirmOrderChangesInput) (composer.StepResponse[any], error) {
		var data []any
		var currentChanges []*dal.Record
		for _, c := range in.Changes {
			u := dal.NewRecord()
			u.Set("id", get(c, "id"))
			u.Set("confirmed_by", in.ConfirmedBy)
			data = append(data, u)
			current := dal.NewRecord()
			current.Set("id", get(c, "id"))
			current.Set("confirmed_by", in.ConfirmedBy)
			current.Set("order_id", in.OrderID)
			current.Set("status", get(c, "status"))
			currentChanges = append(currentChanges, current)
		}
		if withResult {
			res, err := ctx.Container.Resolve(utilsmodulessdk.ORDER).(interface {
				ConfirmOrderChangeResult(ctx context.Context, data []any) (*dal.Record, error)
			}).ConfirmOrderChangeResult(ctx.Context, data)
			if err != nil {
				return composer.StepResponse[any]{}, err
			}
			return composer.NewStepResponse[any](res, any(currentChanges)), nil
		}
		if err := ctx.Container.Resolve(utilsmodulessdk.ORDER).(confirmer).ConfirmOrderChangeRecords(ctx.Context, data); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(currentChanges)), nil
	}
}

// ConfirmOrderChangesStep mirrors confirmOrderChanges: its output is the
// confirmOrderChange result.
var ConfirmOrderChangesStep = composer.CreateStep("confirm-order-changes", confirmOrderChangesInvoke(true), undoConfirmOrderChanges)

// ConfirmOrderChangesWithResultStep is confirmOrderChanges with its output:
// the confirmOrderChange result.
var ConfirmOrderChangesWithResultStep = composer.CreateStep("confirm-order-changes", confirmOrderChangesInvoke(true), undoConfirmOrderChanges)

// undoConfirmOrderChanges is the compensation of confirmOrderChanges.
var undoConfirmOrderChanges = func(ctx composer.StepExecutionContext, data any) error {
	currentChanges, _ := data.([]*dal.Record)
	if len(currentChanges) == 0 {
		return nil
	}
	svc := ctx.Container.Resolve(utilsmodulessdk.ORDER).(interface {
		UndoLastChange(ctx context.Context, orderID string, lastOrderChange *dal.Record) error
	})
	return svc.UndoLastChange(ctx.Context, fmt.Sprint(get(currentChanges[0], "order_id")), currentChanges[0])
}

type decliner interface {
	Internal(className string) *utilsmodulessdk.InternalService
	DeclineOrderChange(ctx context.Context, data []any) error
}

// DeclineOrderChangeStep mirrors declineOrderChangeStep: the order change is
// read before it is declined (the fields of the input, metadata as an object
// field, plus the fields the decline sets) and the compensation writes it
// back (updateOrderChanges).
var DeclineOrderChangeStep = composer.CreateStep("decline-order-change",
	func(ctx composer.StepExecutionContext, data *dal.Record) (composer.StepResponse[any], error) {
		svc, _ := ctx.Container.Resolve(utilsmodulessdk.ORDER).(decliner)
		selects, relations := common.GetSelectsAndRelationsFromObjectArray([]common.ObjectLike{data}, common.SelectsAndRelationsOptions{ObjectFields: []string{"metadata"}}, "")
		selects = append(selects, "order_id", "return_id", "claim_id", "exchange_id", "version", "declined_at", "declined_by", "declined_reason")
		before, err := svc.Internal("OrderChange").Retrieve(ctx.Context, get(data, "id"), utilsmodulessdk.FindConfig{Select: dedupe(selects), Relations: relations})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := svc.DeclineOrderChange(ctx.Context, []any{data}); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(before)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		before, _ := data.(*dal.Record)
		if before == nil {
			return nil
		}
		svc, _ := ctx.Container.Resolve(utilsmodulessdk.ORDER).(decliner)
		_, err := svc.Internal("OrderChange").Update(ctx.Context, before)
		return err
	})

type canceler interface {
	Internal(className string) *utilsmodulessdk.InternalService
	CancelOrderChange(ctx context.Context, data []any) error
}

// CancelOrderChangeStep mirrors cancelOrderChangeStep: the order change is
// read before it is canceled (the fields of the input, metadata as an
// object field, plus the fields the cancel sets) and the compensation
// writes it back (updateOrderChanges).
var CancelOrderChangeStep = composer.CreateStep("cancel-order-change",
	func(ctx composer.StepExecutionContext, data *dal.Record) (composer.StepResponse[any], error) {
		svc, _ := ctx.Container.Resolve(utilsmodulessdk.ORDER).(canceler)
		selects, relations := common.GetSelectsAndRelationsFromObjectArray([]common.ObjectLike{data}, common.SelectsAndRelationsOptions{ObjectFields: []string{"metadata"}}, "")
		selects = append(selects, "order_id", "return_id", "claim_id", "exchange_id", "version", "canceled_at", "canceled_by")
		before, err := svc.Internal("OrderChange").Retrieve(ctx.Context, get(data, "id"), utilsmodulessdk.FindConfig{Select: dedupe(selects), Relations: relations})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := svc.CancelOrderChange(ctx.Context, []any{data}); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(before)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		before, _ := data.(*dal.Record)
		if before == nil {
			return nil
		}
		svc, _ := ctx.Container.Resolve(utilsmodulessdk.ORDER).(canceler)
		_, err := svc.Internal("OrderChange").Update(ctx.Context, before)
		return err
	})

// dedupe mirrors deduplicate (first occurrence kept).
func dedupe(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
