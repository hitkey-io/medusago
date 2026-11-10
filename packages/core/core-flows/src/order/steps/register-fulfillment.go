package steps

import (
	"context"
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type registrar interface {
	RegisterFulfillment(ctx context.Context, data *dal.Record) error
	RegisterShipment(ctx context.Context, data *dal.Record) error
	RegisterDelivery(ctx context.Context, data *dal.Record) error
	CancelFulfillment(ctx context.Context, data *dal.Record) error
}

func registrarOf(c composer.Container) registrar { return c.Resolve(utilsmodulessdk.ORDER).(registrar) }

// revertLastVersion is the compensation of the register steps (the order
// module's revertLastVersion is not ported yet).
func revertLastVersion(ctx composer.StepExecutionContext, orderID any) error {
	if orderID != nil {
		slog.Error("revertLastVersion is not supported yet")
	}
	return nil
}

func registerStep(name string, fn func(r registrar, ctx context.Context, data *dal.Record) error) *composer.Step[*dal.Record, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, data *dal.Record) (composer.StepResponse[any], error) {
			if err := fn(registrarOf(ctx.Container), ctx.Context, data); err != nil {
				return composer.StepResponse[any]{}, err
			}
			return composer.NewStepResponse[any](nil, get(data, "order_id")), nil
		}, revertLastVersion)
}

// RegisterOrderFulfillmentStep mirrors registerOrderFulfillmentStep.
var RegisterOrderFulfillmentStep = registerStep("register-order-fullfillment", func(r registrar, ctx context.Context, d *dal.Record) error { return r.RegisterFulfillment(ctx, d) })

// RegisterOrderShipmentStep mirrors registerOrderShipmentStep.
var RegisterOrderShipmentStep = registerStep("register-order-shipment", func(r registrar, ctx context.Context, d *dal.Record) error { return r.RegisterShipment(ctx, d) })

// RegisterOrderDeliveryStep mirrors registerOrderDeliveryStep.
var RegisterOrderDeliveryStep = registerStep("register-order-delivery", func(r registrar, ctx context.Context, d *dal.Record) error { return r.RegisterDelivery(ctx, d) })

// CancelOrderFulfillmentStep mirrors cancelOrderFulfillmentStep.
var CancelOrderFulfillmentStep = registerStep("cancel-order-fulfillment", func(r registrar, ctx context.Context, d *dal.Record) error { return r.CancelFulfillment(ctx, d) })
