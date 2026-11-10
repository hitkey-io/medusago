package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreatePropertyLabelsStepInput mirrors CreatePropertyLabelsStepInput
// ({ entity, property, label, description? } records).
type CreatePropertyLabelsStepInput struct {
	PropertyLabels []*dal.Record
}

// CreatePropertyLabelsStep mirrors createPropertyLabelsStep (the
// compensation deletes the created labels).
var CreatePropertyLabelsStep = composer.CreateStep("create-property-labels",
	func(ctx composer.StepExecutionContext, in CreatePropertyLabelsStepInput) (composer.StepResponse[[]*dal.Record], error) {
		if len(in.PropertyLabels) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]any{})), nil
		}
		created, err := settingsModule(ctx.Container).CreatePropertyLabels(ctx.Context, in.PropertyLabels)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		ids := make([]any, len(created))
		for i, c := range created {
			ids[i], _ = c.Get("id")
		}
		return composer.NewStepResponse(created, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		return settingsModule(ctx.Container).DeletePropertyLabels(ctx.Context, ids)
	})

// UpdatePropertyLabelsStepInput mirrors UpdatePropertyLabelsStepInput
// ({ id, label?, description? } records).
type UpdatePropertyLabelsStepInput struct {
	PropertyLabels []*dal.Record
}

// UpdatePropertyLabelsStep mirrors updatePropertyLabelsStep: the
// compensation restores the previous label and description.
var UpdatePropertyLabelsStep = composer.CreateStep("update-property-labels",
	func(ctx composer.StepExecutionContext, in UpdatePropertyLabelsStepInput) (composer.StepResponse[[]*dal.Record], error) {
		if len(in.PropertyLabels) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
		}
		svc := settingsModule(ctx.Container)
		ids := make([]any, len(in.PropertyLabels))
		for i, p := range in.PropertyLabels {
			ids[i], _ = p.Get("id")
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		existing, err := svc.ListPropertyLabels(ctx.Context, f)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		previous := make([]*dal.Record, len(existing))
		for i, e := range existing {
			p := dal.NewRecord()
			id, _ := e.Get("id")
			label, _ := e.Get("label")
			p.Set("id", id)
			p.Set("label", label)
			if d, _ := e.Get("description"); d != nil {
				p.Set("description", d)
			}
			previous[i] = p
		}
		updated, err := svc.UpdatePropertyLabels(ctx.Context, in.PropertyLabels)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(updated, any(previous)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		previous, _ := data.([]*dal.Record)
		if len(previous) == 0 {
			return nil
		}
		_, err := settingsModule(ctx.Container).UpdatePropertyLabels(ctx.Context, previous)
		return err
	})

// DeletePropertyLabelsStepInput mirrors DeletePropertyLabelsStepInput.
type DeletePropertyLabelsStepInput struct {
	IDs []any
}

// DeletePropertyLabelsStep mirrors deletePropertyLabelsStep: the
// compensation recreates the deleted labels.
var DeletePropertyLabelsStep = composer.CreateStep("delete-property-labels",
	func(ctx composer.StepExecutionContext, in DeletePropertyLabelsStepInput) (composer.StepResponse[any], error) {
		if len(in.IDs) == 0 {
			return composer.NewStepResponse[any](nil, any([]*dal.Record{})), nil
		}
		svc := settingsModule(ctx.Container)
		f := dal.NewRecord()
		f.Set("id", in.IDs)
		existing, err := svc.ListPropertyLabels(ctx.Context, f)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := svc.DeletePropertyLabels(ctx.Context, in.IDs); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(existing)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		previous, _ := data.([]*dal.Record)
		if len(previous) == 0 {
			return nil
		}
		restore := make([]*dal.Record, len(previous))
		for i, label := range previous {
			r := dal.NewRecord()
			for _, k := range []string{"entity", "property", "label"} {
				v, _ := label.Get(k)
				r.Set(k, v)
			}
			if d, _ := label.Get("description"); d != nil {
				r.Set("description", d)
			}
			restore[i] = r
		}
		_, err := settingsModule(ctx.Container).CreatePropertyLabels(ctx.Context, restore)
		return err
	})
