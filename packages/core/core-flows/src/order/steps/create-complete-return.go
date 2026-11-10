package steps

import (
	"context"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type completeReturnOps interface {
	CreateReturn(ctx context.Context, data *dal.Record) (*dal.Record, error)
	ReceiveReturn(ctx context.Context, data *dal.Record) (*dal.Record, error)
}

func completeReturnsOf(c composer.Container) completeReturnOps {
	return c.Resolve(utilsmodulessdk.ORDER).(completeReturnOps)
}

// CreateCompleteReturnStep mirrors createCompleteReturnStep.
var CreateCompleteReturnStep = composer.CreateStep("create-complete-return",
	func(ctx composer.StepExecutionContext, data *dal.Record) (composer.StepResponse[*dal.Record], error) {
		in := dal.NewRecord()
		in.Set("requested_at", time.Now())
		for _, k := range data.Keys() {
			v, _ := data.Get(k)
			in.Set(k, v)
		}
		created, err := completeReturnsOf(ctx.Container).CreateReturn(ctx.Context, in)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		orderID, _ := data.Get("order_id")
		return composer.NewStepResponse(created, orderID), nil
	},
	func(ctx composer.StepExecutionContext, orderID any) error {
		if orderID == nil {
			return nil
		}
		return RevertLastVersion(ctx, orderID)
	})

// ReceiveReturnStep mirrors receiveReturnStep.
var ReceiveReturnStep = composer.CreateStep("receive-return",
	func(ctx composer.StepExecutionContext, data *dal.Record) (composer.StepResponse[*dal.Record], error) {
		items, _ := data.Get("items")
		if list, _ := items.([]any); len(list) == 0 {
			return composer.NewStepResponse[*dal.Record](nil), nil
		}
		received, err := completeReturnsOf(ctx.Container).ReceiveReturn(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		returnID, _ := data.Get("return_id")
		return composer.NewStepResponse(received, returnID), nil
	},
	func(ctx composer.StepExecutionContext, orderID any) error {
		if orderID == nil {
			return nil
		}
		return RevertLastVersion(ctx, orderID)
	})
