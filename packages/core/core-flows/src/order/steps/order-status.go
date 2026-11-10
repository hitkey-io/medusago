package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type statusOps interface {
	ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error)
	Cancel(ctx context.Context, ids []any) ([]*dal.Record, error)
	CompleteOrder(ctx context.Context, ids []any) ([]*dal.Record, error)
	Archive(ctx context.Context, ids []any) ([]*dal.Record, error)
	UpdateOrders(ctx context.Context, data []*dal.Record) error
}

func statusOf(c composer.Container) statusOps { return c.Resolve(utilsmodulessdk.ORDER).(statusOps) }

func previousStatuses(ctx composer.StepExecutionContext, ids []any) (map[any]any, error) {
	f := dal.NewRecord()
	f.Set("id", ids)
	orders, err := statusOf(ctx.Container).ListBySuffix(ctx.Context, "Orders", f, utilsmodulessdk.FindConfig{Select: []string{"id", "status"}})
	if err != nil {
		return nil, err
	}
	out := map[any]any{}
	for _, o := range orders {
		out[get(o, "id")] = get(o, "status")
	}
	return out, nil
}

func restoreOrders(ctx composer.StepExecutionContext, data any) error {
	list, _ := data.([]*dal.Record)
	if len(list) == 0 {
		return nil
	}
	return statusOf(ctx.Container).UpdateOrders(ctx.Context, list)
}

// CancelOrdersStep mirrors cancelOrdersStep.
var CancelOrdersStep = composer.CreateStep("cancel-orders",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[[]*dal.Record], error) {
		prev, err := previousStatuses(ctx, ids)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		canceled, err := statusOf(ctx.Container).Cancel(ctx.Context, ids)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		var comp []*dal.Record
		for _, o := range canceled {
			r := dal.NewRecord()
			r.Set("id", get(o, "id"))
			r.Set("status", prev[get(o, "id")])
			r.Set("canceled_at", nil)
			r.Set("canceled_by", nil)
			comp = append(comp, r)
		}
		return composer.NewStepResponse(canceled, any(comp)), nil
	}, restoreOrders)

// CompleteOrdersStep mirrors completeOrdersStep.
var CompleteOrdersStep = composer.CreateStep("complete-orders",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[[]*dal.Record], error) {
		prev, err := previousStatuses(ctx, ids)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		completed, err := statusOf(ctx.Container).CompleteOrder(ctx.Context, ids)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		var comp []*dal.Record
		for _, o := range completed {
			r := dal.NewRecord()
			r.Set("id", get(o, "id"))
			r.Set("status", prev[get(o, "id")])
			comp = append(comp, r)
		}
		return composer.NewStepResponse(completed, any(comp)), nil
	}, restoreOrders)

// ArchiveOrdersStep mirrors archiveOrdersStep.
var ArchiveOrdersStep = composer.CreateStep("archive-orders",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[[]*dal.Record], error) {
		archived, err := statusOf(ctx.Container).Archive(ctx.Context, ids)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		var comp []*dal.Record
		for _, o := range archived {
			r := dal.NewRecord()
			r.Set("id", get(o, "id"))
			r.Set("status", get(o, "status"))
			comp = append(comp, r)
		}
		return composer.NewStepResponse(archived, any(comp)), nil
	}, restoreOrders)
