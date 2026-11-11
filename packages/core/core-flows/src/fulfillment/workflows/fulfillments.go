package workflows

import (
	"context"
	lockingsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/locking/steps"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type fulfillmentService interface {
	Internal(className string) *utilsmodulessdk.InternalService
	CreateFulfillment(ctx context.Context, data *dal.Record) (*dal.Record, error)
	CreateReturnFulfillment(ctx context.Context, data *dal.Record) (*dal.Record, error)
	UpdateFulfillment(ctx context.Context, id any, data *dal.Record) (*dal.Record, error)
	CancelFulfillment(ctx context.Context, id any) (*dal.Record, error)
}

func fulfillmentOf(c composer.Container) fulfillmentService {
	return c.Resolve(utilsmodulessdk.FULFILLMENT).(fulfillmentService)
}

func getv(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

// CreateFulfillmentStep mirrors createFulfillmentStep.
var CreateFulfillmentStep = composer.CreateStep("create-fulfillment",
	func(ctx composer.StepExecutionContext, data *dal.Record) (composer.StepResponse[*dal.Record], error) {
		f, err := fulfillmentOf(ctx.Container).CreateFulfillment(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(f, getv(f, "id")), nil
	},
	func(ctx composer.StepExecutionContext, id any) error {
		if id == nil {
			return nil
		}
		_, err := fulfillmentOf(ctx.Container).CancelFulfillment(ctx.Context, id)
		return err
	})

// CreateFulfillmentWorkflow mirrors createFulfillmentWorkflow.
var CreateFulfillmentWorkflow = composer.CreateWorkflow("create-fulfillment-workflow",
	func(e *composer.Execution, input *dal.Record) (*dal.Record, error) {
		v := dal.NewRecord()
		v.Set("id", getv(input, "location_id"))
		list := false
		res, err := commonsteps.UseRemoteQueryStep.Config("get-location", composer.StepOptions{}).Run(e, commonsteps.RemoteStepInput{
			EntryPoint: "stock_location",
			Fields: []string{"id", "name", "metadata", "created_at", "updated_at", "address.id", "address.address_1", "address.address_2", "address.city",
				"address.country_code", "address.phone", "address.province", "address.postal_code", "address.metadata"},
			Variables: v, List: &list, ThrowIfKeyNotFound: true,
		})
		if err != nil {
			return nil, err
		}
		var location any
		if len(res.Data) > 0 {
			location = res.Data[0]
		}
		in := dal.NewRecord()
		for _, k := range input.Keys() {
			x, _ := input.Get(k)
			in.Set(k, x)
		}
		in.Set("location", location)
		return CreateFulfillmentStep.Run(e, in)
	})

// UpdateFulfillmentStep mirrors updateFulfillmentStep.
var UpdateFulfillmentStep = composer.CreateStep("update-fulfillment",
	func(ctx composer.StepExecutionContext, input *dal.Record) (composer.StepResponse[*dal.Record], error) {
		svc := fulfillmentOf(ctx.Container)
		id := getv(input, "id")
		data := dal.NewRecord()
		var sel []string
		for _, k := range input.Keys() {
			if k == "id" {
				continue
			}
			x, _ := input.Get(k)
			data.Set(k, x)
			sel = append(sel, k)
		}
		before, err := svc.Internal("Fulfillment").Retrieve(ctx.Context, id, utilsmodulessdk.FindConfig{Select: sel})
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		updated, err := svc.UpdateFulfillment(ctx.Context, id, data)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(updated, any(before)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		before, _ := data.(*dal.Record)
		if before == nil {
			return nil
		}
		d := dal.NewRecord()
		for _, k := range before.Keys() {
			if k == "id" {
				continue
			}
			x, _ := before.Get(k)
			d.Set(k, x)
		}
		_, err := fulfillmentOf(ctx.Container).UpdateFulfillment(ctx.Context, getv(before, "id"), d)
		return err
	})

// UpdateFulfillmentWorkflow mirrors updateFulfillmentWorkflow.
var UpdateFulfillmentWorkflow = composer.CreateWorkflow("update-fulfillment-workflow",
	func(e *composer.Execution, input *dal.Record) (*dal.Record, error) {
		return UpdateFulfillmentStep.Run(e, input)
	})

func notAllowed(msg string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, msg)
}

// ValidateShipmentStep mirrors validateShipmentStep.
var ValidateShipmentStep = composer.CreateStep("validate-shipment",
	func(ctx composer.StepExecutionContext, id any) (composer.StepResponse[any], error) {
		f, err := fulfillmentOf(ctx.Container).Internal("Fulfillment").Retrieve(ctx.Context, id, utilsmodulessdk.FindConfig{Select: []string{"shipped_at", "canceled_at", "shipping_option_id"}})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		if getv(f, "shipped_at") != nil {
			return composer.StepResponse[any]{}, notAllowed("Shipment has already been created")
		}
		if getv(f, "canceled_at") != nil {
			return composer.StepResponse[any]{}, notAllowed("Cannot create shipment for a canceled fulfillment")
		}
		if getv(f, "shipping_option_id") == nil {
			return composer.StepResponse[any]{}, notAllowed("Cannot create shipment without a Shipping Option")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// CreateShipmentWorkflow mirrors createShipmentWorkflow.
var CreateShipmentWorkflow = composer.CreateWorkflow("create-shipment-workflow",
	func(e *composer.Execution, input *dal.Record) (*dal.Record, error) {
		if _, err := ValidateShipmentStep.Run(e, getv(input, "id")); err != nil {
			return nil, err
		}
		u := dal.NewRecord()
		for _, k := range input.Keys() {
			x, _ := input.Get(k)
			u.Set(k, x)
		}
		u.Set("shipped_at", dal.Now())
		return UpdateFulfillmentWorkflow.RunAsStep(e, u)
	})

// ValidateFulfillmentDeliverabilityStep mirrors validateFulfillmentDeliverabilityStep.
var ValidateFulfillmentDeliverabilityStep = composer.CreateStep("validate-fulfillment-deliverability",
	func(ctx composer.StepExecutionContext, f *dal.Record) (composer.StepResponse[any], error) {
		if getv(f, "canceled_at") != nil {
			return composer.StepResponse[any]{}, notAllowed("Cannot deliver an already canceled fulfillment")
		}
		if getv(f, "delivered_at") != nil {
			return composer.StepResponse[any]{}, notAllowed("Fulfillment has already been marked delivered")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// MarkFulfillmentAsDeliveredWorkflow mirrors markFulfillmentAsDeliveredWorkflow.
var MarkFulfillmentAsDeliveredWorkflow = composer.CreateWorkflow("mark-fulfillment-as-delivered-workflow",
	func(e *composer.Execution, id any) (*dal.Record, error) {
		key, _ := id.(string)
		if _, err := lockingsteps.AcquireLockStep.Run(e, lockingsteps.AcquireLockStepInput{Key: lockingsteps.Key(key), Timeout: 2, TTL: 10}); err != nil {
			return nil, err
		}
		v := dal.NewRecord()
		v.Set("id", id)
		list := false
		res, err := commonsteps.UseRemoteQueryStep.Run(e, commonsteps.RemoteStepInput{EntryPoint: "fulfillment", Fields: []string{"id", "delivered_at", "canceled_at"}, Variables: v, ThrowIfKeyNotFound: true, List: &list})
		if err != nil {
			return nil, err
		}
		var f *dal.Record
		if len(res.Data) > 0 {
			f, _ = res.Data[0].(*dal.Record)
		}
		if _, err := ValidateFulfillmentDeliverabilityStep.Run(e, f); err != nil {
			return nil, err
		}
		u := dal.NewRecord()
		u.Set("id", id)
		u.Set("delivered_at", dal.Now())
		updated, err := UpdateFulfillmentWorkflow.RunAsStep(e, u)
		if err != nil {
			return nil, err
		}
		if _, err := lockingsteps.ReleaseLockStep.Run(e, lockingsteps.ReleaseLockStepInput{Key: lockingsteps.Key(key)}); err != nil {
			return nil, err
		}
		return updated, nil
	})

// CancelFulfillmentStep mirrors cancelFulfillmentStep.
var CancelFulfillmentStep = composer.CreateStep("cancel-fulfillment",
	func(ctx composer.StepExecutionContext, id any) (composer.StepResponse[any], error) {
		if _, err := fulfillmentOf(ctx.Container).CancelFulfillment(ctx.Context, id); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// CancelFulfillmentWorkflow mirrors cancelFulfillmentWorkflow.
var CancelFulfillmentWorkflow = composer.CreateWorkflow("cancel-fulfillment-workflow",
	func(e *composer.Execution, id any) (any, error) {
		_, err := CancelFulfillmentStep.Run(e, id)
		return nil, err
	})

// CreateReturnFulfillmentStep mirrors createReturnFulfillmentStep.
var CreateReturnFulfillmentStep = composer.CreateStep("create-return-fulfillment",
	func(ctx composer.StepExecutionContext, data *dal.Record) (composer.StepResponse[*dal.Record], error) {
		f, err := fulfillmentOf(ctx.Container).CreateReturnFulfillment(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(f, getv(f, "id")), nil
	},
	func(ctx composer.StepExecutionContext, id any) error {
		if id == nil {
			return nil
		}
		_, err := fulfillmentOf(ctx.Container).CancelFulfillment(ctx.Context, id)
		return err
	})

// CreateReturnFulfillmentWorkflow mirrors createReturnFulfillmentWorkflow.
var CreateReturnFulfillmentWorkflow = composer.CreateWorkflow("create-return-fulfillment-workflow",
	func(e *composer.Execution, input *dal.Record) (*dal.Record, error) {
		v := dal.NewRecord()
		v.Set("id", getv(input, "location_id"))
		list := false
		res, err := commonsteps.UseRemoteQueryStep.Config("get-location", composer.StepOptions{}).Run(e, commonsteps.RemoteStepInput{
			EntryPoint: "stock_location",
			Fields: []string{"id", "name", "metadata", "created_at", "updated_at", "address.id", "address.address_1", "address.address_2", "address.city",
				"address.country_code", "address.phone", "address.province", "address.postal_code", "address.metadata"},
			Variables: v, List: &list, ThrowIfKeyNotFound: true,
		})
		if err != nil {
			return nil, err
		}
		var location any
		if len(res.Data) > 0 {
			location = res.Data[0]
		}
		in := dal.NewRecord()
		for _, k := range input.Keys() {
			x, _ := input.Get(k)
			in.Set(k, x)
		}
		in.Set("location", location)
		return CreateReturnFulfillmentStep.Run(e, in)
	})
