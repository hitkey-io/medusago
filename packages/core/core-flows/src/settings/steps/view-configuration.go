package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateViewConfigurationStep mirrors createViewConfigurationStep (the
// compensation deletes the created configuration).
var CreateViewConfigurationStep = composer.CreateStep("create-view-configuration",
	func(ctx composer.StepExecutionContext, data *dal.Record) (composer.StepResponse[*dal.Record], error) {
		created, err := settingsModule(ctx.Container).CreateViewConfigurations(ctx.Context, []*dal.Record{data})
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		var out *dal.Record
		if len(created) > 0 {
			out = created[0]
		}
		var id any
		if out != nil {
			id, _ = out.Get("id")
		}
		return composer.NewStepResponse(out, any(id)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		if data == nil || data == "" {
			return nil
		}
		return settingsModule(ctx.Container).DeleteViewConfigurations(ctx.Context, []any{data})
	})

// SetActiveViewConfigurationStepInput mirrors SetActiveViewConfigurationStepInput.
type SetActiveViewConfigurationStepInput struct {
	ID     string
	Entity string
	UserID string
}

type activeViewCompensation struct {
	entity, userID       string
	previousActiveViewID string
}

// SetActiveViewConfigurationStep mirrors setActiveViewConfigurationStep:
// the compensation restores the previously active view (or clears it).
var SetActiveViewConfigurationStep = composer.CreateStep("set-active-view-configuration",
	func(ctx composer.StepExecutionContext, in SetActiveViewConfigurationStepInput) (composer.StepResponse[string], error) {
		svc := settingsModule(ctx.Container)
		current, err := svc.GetActiveViewConfiguration(ctx.Context, in.Entity, in.UserID)
		if err != nil {
			return composer.StepResponse[string]{}, err
		}
		if err := svc.SetActiveViewConfiguration(ctx.Context, in.Entity, in.UserID, in.ID); err != nil {
			return composer.StepResponse[string]{}, err
		}
		comp := activeViewCompensation{entity: in.Entity, userID: in.UserID}
		if current != nil {
			if id, ok := current.Get("id"); ok {
				comp.previousActiveViewID, _ = id.(string)
			}
		}
		return composer.NewStepResponse(in.ID, any(comp)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		comp, ok := data.(activeViewCompensation)
		if !ok {
			return nil
		}
		svc := settingsModule(ctx.Container)
		if comp.previousActiveViewID != "" {
			return svc.SetActiveViewConfiguration(ctx.Context, comp.entity, comp.userID, comp.previousActiveViewID)
		}
		return svc.ClearActiveViewConfiguration(ctx.Context, comp.entity, comp.userID)
	})

// UpdateViewConfigurationStepInput mirrors UpdateViewConfigurationStepInput.
type UpdateViewConfigurationStepInput struct {
	ID   string
	Data *dal.Record
}

type viewConfigurationCompensation struct {
	id            string
	previousState *dal.Record
}

// UpdateViewConfigurationStep mirrors updateViewConfigurationStep: the
// compensation restores the previous state (without id and timestamps).
var UpdateViewConfigurationStep = composer.CreateStep("update-view-configuration",
	func(ctx composer.StepExecutionContext, in UpdateViewConfigurationStepInput) (composer.StepResponse[*dal.Record], error) {
		svc := settingsModule(ctx.Container)
		current, err := svc.RetrieveViewConfiguration(ctx.Context, in.ID)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		data := in.Data
		if data == nil {
			data = dal.NewRecord()
		}
		updated, err := svc.UpdateViewConfigurations(ctx.Context, in.ID, data)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(updated, any(viewConfigurationCompensation{id: in.ID, previousState: current})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		comp, ok := data.(viewConfigurationCompensation)
		if !ok || comp.id == "" || comp.previousState == nil {
			return nil
		}
		restore := dal.NewRecord()
		for _, k := range comp.previousState.Keys() {
			if k == "id" || k == "created_at" || k == "updated_at" {
				continue
			}
			v, _ := comp.previousState.Get(k)
			restore.Set(k, v)
		}
		_, err := settingsModule(ctx.Container).UpdateViewConfigurations(ctx.Context, comp.id, restore)
		return err
	})
