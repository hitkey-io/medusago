package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type orderUpdater interface {
	Internal(className string) *utilsmodulessdk.InternalService
	ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error)
	UpdateOrders(ctx context.Context, data []*dal.Record) error
	RegisterOrderChange(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
}

func updaterOf(c composer.Container) orderUpdater { return c.Resolve(utilsmodulessdk.ORDER).(orderUpdater) }

// UpdateOrdersStepInput mirrors UpdateOrdersStepInput.
type UpdateOrdersStepInput struct {
	Selector *dal.Record
	Update   *dal.Record
}

// UpdateOrdersStep mirrors updateOrdersStep: the matching orders are
// updated and returned with their addresses.
var UpdateOrdersStep = composer.CreateStep("update-orders",
	func(ctx composer.StepExecutionContext, in UpdateOrdersStepInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := updaterOf(ctx.Container)
		selects, relations := common.GetSelectsAndRelationsFromObjectArray([]common.ObjectLike{in.Update}, common.SelectsAndRelationsOptions{}, "")
		prev, err := svc.ListBySuffix(ctx.Context, "Orders", in.Selector, utilsmodulessdk.FindConfig{Select: selects, Relations: relations})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		u := dal.NewRecord()
		u.Set("selector", in.Selector)
		u.Set("data", in.Update)
		if _, err := svc.Internal("Order").Update(ctx.Context, u); err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		updated, err := svc.Internal("Order").List(ctx.Context, in.Selector, utilsmodulessdk.FindConfig{Relations: []string{"shipping_address", "billing_address"}})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(updated, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		return updaterOf(ctx.Container).UpdateOrders(ctx.Context, prev)
	})

// RegisterOrderChangesStep mirrors registerOrderChangesStep.
var RegisterOrderChangesStep = composer.CreateStep("register-order-change",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[any], error) {
		if len(data) == 0 {
			return composer.NewStepResponse[any](nil, any([]any{})), nil
		}
		changes, err := updaterOf(ctx.Container).RegisterOrderChange(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(ids(changes))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		list, _ := data.([]any)
		if len(list) == 0 {
			return nil
		}
		_, err := updaterOf(ctx.Container).Internal("OrderChange").Delete(ctx.Context, list)
		return err
	})
