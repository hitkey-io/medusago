// Package workflows mirrors @medusajs/core-flows/src/shipping-options/workflows.
package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	fulfillmentsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func idEvents(ids []any) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		ev := dal.NewRecord()
		ev.Set("id", id)
		out[i] = ev
	}
	return out
}

func hook(e *composer.Execution, name string, kv ...any) error {
	r := dal.NewRecord()
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i].(string), kv[i+1])
	}
	return composer.CreateHook(e, name, r)
}

// CreateShippingOptionTypesStep mirrors createShippingOptionTypesStep.
var CreateShippingOptionTypesStep = commonsteps.CreateRecordsStep("create-shipping-option-types", utilsmodulessdk.FULFILLMENT, "ShippingOptionType")

// DeleteShippingOptionTypesStep mirrors deleteShippingOptionTypesStep.
var DeleteShippingOptionTypesStep = commonsteps.SoftDeleteRecordsStep("delete-shipping-option-types", utilsmodulessdk.FULFILLMENT, "ShippingOptionType")

// UpdateShippingOptionTypesStep mirrors updateShippingOptionTypesStep.
var UpdateShippingOptionTypesStep = composer.CreateStep("update-shipping-option-types",
	func(ctx composer.StepExecutionContext, in fulfillmentsteps.SelectorUpdateInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := fulfillmentsteps.Fulfillment(ctx.Container)
		prev, err := commonsteps.PrevDataFor(ctx.Context, svc, "ShippingOptionTypes", in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		out, err := svc.UpdateShippingOptionTypes(ctx.Context, in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(out, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		_, err := fulfillmentsteps.Fulfillment(ctx.Container).Internal("ShippingOptionType").Upsert(ctx.Context, prev)
		return err
	})

// CreateShippingOptionTypesInput mirrors CreateShippingOptionTypesWorkflowInput.
type CreateShippingOptionTypesInput struct {
	ShippingOptionTypes []*dal.Record
	AdditionalData      *dal.Record
}

// CreateShippingOptionTypesWorkflow mirrors createShippingOptionTypesWorkflow.
var CreateShippingOptionTypesWorkflow = composer.CreateWorkflow("create-shipping-option-types",
	func(e *composer.Execution, input CreateShippingOptionTypesInput) ([]*dal.Record, error) {
		created, err := CreateShippingOptionTypesStep.Run(e, input.ShippingOptionTypes)
		if err != nil {
			return nil, err
		}
		if err := hook(e, "shippingOptionTypesCreated", "shipping_option_types", created, "additional_data", input.AdditionalData); err != nil {
			return nil, err
		}
		_, err = commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "shipping-option-type.created", Data: idEvents(commonsteps.RecordIDs(created))})
		return created, err
	})

// UpdateShippingOptionTypesInput mirrors UpdateShippingOptionTypesWorkflowInput.
type UpdateShippingOptionTypesInput struct {
	Selector       *dal.Record
	Update         *dal.Record
	AdditionalData *dal.Record
}

// UpdateShippingOptionTypesWorkflow mirrors updateShippingOptionTypesWorkflow.
var UpdateShippingOptionTypesWorkflow = composer.CreateWorkflow("update-shipping-option-types",
	func(e *composer.Execution, input UpdateShippingOptionTypesInput) ([]*dal.Record, error) {
		updated, err := UpdateShippingOptionTypesStep.Run(e, fulfillmentsteps.SelectorUpdateInput{Selector: input.Selector, Update: input.Update})
		if err != nil {
			return nil, err
		}
		if err := hook(e, "shippingOptionTypesUpdated", "shipping_option_types", updated, "additional_data", input.AdditionalData); err != nil {
			return nil, err
		}
		_, err = commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "shipping-option-type.updated", Data: idEvents(commonsteps.RecordIDs(updated))})
		return updated, err
	})

var validateDeleteShippingOptionTypesStep = composer.CreateStep("validate-delete-shipping-option-types",
	func(ctx composer.StepExecutionContext, options []any) (composer.StepResponse[any], error) {
		if len(options) > 0 {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Cannot delete shipping option type because some shipping options are using it.")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// DeleteShippingOptionTypesWorkflow mirrors deleteShippingOptionTypesWorkflow.
var DeleteShippingOptionTypesWorkflow = composer.CreateWorkflow("delete-shipping-option-types",
	func(e *composer.Execution, ids []any) (any, error) {
		f := dal.NewRecord()
		f.Set("shipping_option_type_id", ids)
		p := dal.NewRecord()
		p.Set("take", float64(1))
		q, err := commonsteps.UseQueryGraphStep.Config("get-shipping-options", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "shipping_option", Filters: f, Pagination: p, Fields: []string{"id"}},
		})
		if err != nil {
			return nil, err
		}
		options, _ := q.Data.([]any)
		if _, err := validateDeleteShippingOptionTypesStep.Run(e, options); err != nil {
			return nil, err
		}
		if _, err := DeleteShippingOptionTypesStep.Run(e, ids); err != nil {
			return nil, err
		}
		if err := hook(e, "shippingOptionTypesDeleted", "ids", ids); err != nil {
			return nil, err
		}
		keys := dal.NewRecord()
		keys.Set("shipping_option_type_id", ids)
		entry := dal.NewRecord()
		entry.Set(utilsmodulessdk.FULFILLMENT, keys)
		return nil, composer.Parallelize(
			func() error { _, err := commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{entry}); return err },
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "shipping-option-type.deleted", Data: idEvents(ids)})
				return err
			},
		)
	})
