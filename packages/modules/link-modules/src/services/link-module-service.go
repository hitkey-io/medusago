// Package services mirrors @medusajs/link-modules/src/services.
package services

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/link-modules/src/utils"
)

// LinkModuleService mirrors LinkModuleService: the service of one link
// module, reading the rows of its link table.
type LinkModuleService struct {
	Config     *joiner.ModuleJoinerConfig
	repository *dal.Repository
	events     utilsmodulessdk.ModuleEventSink
}

// SetEventBus sets the event bus the attached / detached events are
// emitted to (eventBusModuleService_).
func (s *LinkModuleService) SetEventBus(bus eventbus.Emitter) { s.events.SetEventBus(bus) }

// emitLinkEvent mirrors the moduleEventBuilderFactory call of the link
// module service methods (@EmitEvents): `<entityName>.<action>` with the
// ids of the links, from source serviceName and object entityName.
func (s *LinkModuleService) emitLinkEvent(ctx context.Context, action string, ids []any) {
	entity := s.repository.Entity.Meta.ClassName
	ev := utilsmodulessdk.ModuleEventBuilderFactory(s.Config.ServiceName, action, entity, entity+"."+action)(ids...)
	if ev != nil {
		s.events.EmitModuleEvents(ctx, []dal.ReadyEvent{*ev})
	}
}

func rowIDs(rows []dal.Row) []any {
	ids := make([]any, len(rows))
	for i, r := range rows {
		ids[i] = r["id"]
	}
	return ids
}

// NewLinkModuleService builds the service of a link module.
func NewLinkModuleService(db dal.Querier, cfg *joiner.ModuleJoinerConfig) (*LinkModuleService, error) {
	if len(cfg.Relationships) != 2 {
		return nil, fmt.Errorf("link %s must have exactly 2 relationships", cfg.ServiceName)
	}
	entity, err := utils.GenerateEntity(cfg, cfg.Relationships[0], cfg.Relationships[1])
	if err != nil {
		return nil, err
	}
	className := cfg.ServiceName
	if len(cfg.Alias) > 0 && cfg.Alias[0].Entity != "" {
		className = cfg.Alias[0].Entity
	}
	model := entity.ToDMLEntity(className)
	return &LinkModuleService{
		Config:     cfg,
		repository: &dal.Repository{DB: db, Schema: dal.NewSchema(nil), Entity: model},
	}, nil
}

func findOptions(filters *dal.Record, config utilsmodulessdk.FindConfig) dal.FindOptions {
	return dal.FindOptions{
		Where:       filters,
		Select:      config.Select,
		Relations:   config.Relations,
		Order:       config.Order,
		Take:        config.Take,
		Skip:        config.Skip,
		WithDeleted: config.WithDeleted,
	}
}

// List mirrors list: no default ordering and no default limit.
func (s *LinkModuleService) List(ctx context.Context, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	return s.repository.Find(ctx, findOptions(filters, config))
}

// ListAndCount mirrors listAndCount.
func (s *LinkModuleService) ListAndCount(ctx context.Context, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, int, error) {
	return s.repository.FindAndCount(ctx, findOptions(filters, config))
}
