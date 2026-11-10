package steps

import (
	"context"
	"log/slog"

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

// ConfirmOrderChangesStep mirrors confirmOrderChanges (the undoLastChange
// compensation is not ported yet).
var ConfirmOrderChangesStep = composer.CreateStep("confirm-order-changes",
	func(ctx composer.StepExecutionContext, in ConfirmOrderChangesInput) (composer.StepResponse[any], error) {
		var data []any
		for _, c := range in.Changes {
			u := dal.NewRecord()
			u.Set("id", get(c, "id"))
			u.Set("confirmed_by", in.ConfirmedBy)
			data = append(data, u)
		}
		if err := ctx.Container.Resolve(utilsmodulessdk.ORDER).(confirmer).ConfirmOrderChangeRecords(ctx.Context, data); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(in.OrderID)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		if data != nil {
			slog.Error("undoLastChange is not supported yet")
		}
		return nil
	})

// DeclineOrderChangeStep mirrors declineOrderChangeStep (the
// updateOrderChanges compensation is not ported yet).
var DeclineOrderChangeStep = composer.CreateStep("decline-order-change",
	func(ctx composer.StepExecutionContext, data *dal.Record) (composer.StepResponse[any], error) {
		svc, _ := ctx.Container.Resolve(utilsmodulessdk.ORDER).(interface {
			DeclineOrderChange(ctx context.Context, data []any) error
		})
		if err := svc.DeclineOrderChange(ctx.Context, []any{data}); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)
