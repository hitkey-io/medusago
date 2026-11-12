package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// caller is the generated method dispatch of a module service
// (MedusaService.Call: createX, softDeleteX, restoreX, deleteX, ...).
type caller interface {
	Call(ctx context.Context, method string, args []any) (any, error)
}

// EntitiesStepInput mirrors CreateEntitiesStepType / DeleteEntitiesStepType.
type EntitiesStepInput struct {
	ModuleRegistrationName string `json:"moduleRegistrationName"`
	InvokeMethod           string `json:"invokeMethod"`
	CompensateMethod       string `json:"compensateMethod"`
	// EntityIdentifier defaults to "id" (create).
	EntityIdentifier string `json:"entityIdentifier,omitempty"`
	Data             []any  `json:"data"`
}

func entitiesCompensation(identifiers []any, in EntitiesStepInput) *dal.Record {
	r := dal.NewRecord()
	r.Set("entityIdentifiers", identifiers)
	r.Set("moduleRegistrationName", in.ModuleRegistrationName)
	r.Set("compensateMethod", in.CompensateMethod)
	return r
}

func compensateEntities(ctx composer.StepExecutionContext, data any) error {
	r, _ := data.(*dal.Record)
	if r == nil {
		return nil
	}
	ids, _ := r.Get("entityIdentifiers")
	list, _ := ids.([]any)
	if len(list) == 0 {
		return nil
	}
	name, _ := r.Get("moduleRegistrationName")
	method, _ := r.Get("compensateMethod")
	svc, _ := ctx.Container.Resolve(name.(string)).(caller)
	_, err := svc.Call(ctx.Context, method.(string), []any{list})
	return err
}

// CreateEntitiesStep mirrors createEntitiesStep: the module method creates
// the data; the compensation calls compensateMethod with the identifiers of
// the created entities.
var CreateEntitiesStep = composer.CreateStep("create-entities-step",
	func(ctx composer.StepExecutionContext, in EntitiesStepInput) (composer.StepResponse[[]any], error) {
		identifier := in.EntityIdentifier
		if identifier == "" {
			identifier = "id"
		}
		created := []any{}
		if len(in.Data) > 0 {
			svc, _ := ctx.Container.Resolve(in.ModuleRegistrationName).(caller)
			out, err := svc.Call(ctx.Context, in.InvokeMethod, []any{in.Data})
			if err != nil {
				return composer.StepResponse[[]any]{}, err
			}
			switch t := out.(type) {
			case []*dal.Record:
				for _, r := range t {
					created = append(created, r)
				}
			case []any:
				created = t
			case *dal.Record:
				created = append(created, t)
			}
		}
		identifiers := make([]any, 0, len(created))
		for _, c := range created {
			if r, ok := c.(*dal.Record); ok {
				v, _ := r.Get(identifier)
				identifiers = append(identifiers, v)
			}
		}
		return composer.NewStepResponse(created, any(entitiesCompensation(identifiers, in))), nil
	}, compensateEntities)

// DeleteEntitiesStep mirrors deleteEntitiesStep: the module method deletes
// the data; the compensation calls compensateMethod with the same data.
var DeleteEntitiesStep = composer.CreateStep("delete-entities-step",
	func(ctx composer.StepExecutionContext, in EntitiesStepInput) (composer.StepResponse[any], error) {
		if len(in.Data) > 0 {
			svc, _ := ctx.Container.Resolve(in.ModuleRegistrationName).(caller)
			if _, err := svc.Call(ctx.Context, in.InvokeMethod, []any{in.Data}); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		data := in.Data
		if data == nil {
			data = []any{}
		}
		return composer.NewStepResponse[any](nil, any(entitiesCompensation(data, in))), nil
	}, compensateEntities)
