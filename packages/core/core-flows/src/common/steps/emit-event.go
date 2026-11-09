package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// EmitEventStepInput mirrors the input of emitEventStep. Data is one event
// payload or a list of payloads ([]any).
type EmitEventStepInput struct {
	EventName string
	Options   *dal.Record
	Metadata  *dal.Record
	Data      any
}

type emitted struct {
	EventGroupID string
	EventName    string
}

// EmitEventStep mirrors emitEventStep: the events join the workflow's event
// group (released when the workflow succeeds).
var EmitEventStep = composer.CreateStep("emit-event-step",
	func(ctx composer.StepExecutionContext, in EmitEventStepInput) (composer.StepResponse[any], error) {
		if in.Data == nil {
			return composer.NewStepResponse[any](nil), nil
		}
		bus, _ := ctx.Container.Resolve(utilsmodulessdk.EVENT_BUS).(composer.EventBus)
		meta := dal.NewRecord()
		if in.Metadata != nil {
			for _, k := range in.Metadata.Keys() {
				v, _ := in.Metadata.Get(k)
				meta.Set(k, v)
			}
		}
		if ctx.EventGroupID != "" {
			meta.Set("eventGroupId", ctx.EventGroupID)
		}
		list, ok := in.Data.([]any)
		if !ok {
			list = []any{in.Data}
		}
		if len(list) == 0 || bus == nil {
			return composer.NewStepResponse[any](nil), nil
		}
		msgs := make([]composer.Message, len(list))
		for i, d := range list {
			msgs[i] = composer.Message{Name: in.EventName, Data: d, Options: in.Options, Metadata: meta}
		}
		if err := bus.Emit(ctx.Context, msgs); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](emitted{EventGroupID: ctx.EventGroupID, EventName: in.EventName}), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		e, ok := data.(emitted)
		if !ok || e.EventGroupID == "" {
			return nil
		}
		bus, _ := ctx.Container.Resolve(utilsmodulessdk.EVENT_BUS).(composer.EventBus)
		if bus == nil {
			return nil
		}
		return bus.ClearGroupedEvents(ctx.Context, e.EventGroupID, []string{e.EventName})
	})
