package events

import (
	"context"
	"errors"

	typesmodulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// region: service
// BlogModuleService is the main service of a module.
type BlogModuleService struct {
	*utilsmodulessdk.MedusaService
	container typesmodulessdk.Resolver
}

// NewBlogModuleService is the Service function of the module's definition.
// It keeps the container the module is loaded with, to resolve the event
// module when an event is emitted.
func NewBlogModuleService(opts typesmodulessdk.ServiceOptions, base *utilsmodulessdk.MedusaService) (any, error) {
	return &BlogModuleService{MedusaService: base, container: opts.Container}, nil
}

// PerformAction emits a service event.
func (s *BlogModuleService) PerformAction(ctx context.Context) error {
	// TODO perform action

	bus, ok := s.container.Resolve(utilsmodulessdk.EVENT_BUS).(eventbus.Emitter)
	if !ok {
		return errors.New("the event module isn't registered")
	}

	payload := dal.NewRecord()
	payload.Set("id", "123")
	// other data payload

	return bus.Emit(ctx, []eventbus.Message{
		{Name: "custom.event", Data: payload},
	})
}

// endregion: service
