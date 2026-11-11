// Package services mirrors @medusajs/settings/src/services.
package services

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/settings/src/models"
)

// SettingsModuleService mirrors SettingsModuleService (view configurations,
// user preferences, property labels, layout configurations). Entity
// discovery (listDiscoverableEntities, generateEntityColumns) is not ported.
type SettingsModuleService struct {
	*utilsmodulessdk.MedusaService
}

// NewSettingsModuleService builds the service.
func NewSettingsModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*SettingsModuleService, error) {
	return &SettingsModuleService{MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig)}, nil
}

// Of returns the settings module service registered in a container (the
// module service or a bare generated service).
func Of(v any) *SettingsModuleService {
	switch t := v.(type) {
	case *SettingsModuleService:
		return t
	case *utilsmodulessdk.MedusaService:
		return &SettingsModuleService{MedusaService: t}
	}
	return nil
}

// viewConfigurationKeys are the ViewConfiguration properties a create
// accepts (other input keys, e.g. set_active, are not persisted).
var viewConfigurationKeys = map[string]bool{
	"id": true, "entity": true, "name": true, "user_id": true, "is_system_default": true,
	"configuration": true, "created_at": true, "updated_at": true, "deleted_at": true,
}

// Truthy mirrors JS truthiness for JSON-like values.
func Truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	}
	return true
}

// CreateViewConfigurations mirrors createViewConfigurations: a system
// default cannot have a user_id and is unique per entity.
func (s *SettingsModuleService) CreateViewConfigurations(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	svc := s.Internal("ViewConfiguration")
	for _, config := range data {
		isDefault, _ := config.Get("is_system_default")
		userID, _ := config.Get("user_id")
		if Truthy(isDefault) && Truthy(userID) {
			return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "System default view configurations cannot have a user_id")
		}
		if Truthy(isDefault) {
			entity, _ := config.Get("entity")
			f := dal.NewRecord()
			f.Set("entity", entity)
			f.Set("is_system_default", true)
			existing, err := svc.List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id"}})
			if err != nil {
				return nil, err
			}
			if len(existing) > 0 {
				return nil, common.NewMedusaError(common.MedusaErrorTypes.DUPLICATE_ERROR,
					fmt.Sprintf("A system default view configuration already exists for entity: %v", entity))
			}
		}
	}
	input := make([]*dal.Record, len(data))
	for i, d := range data {
		c := dal.NewRecord()
		for _, k := range d.Keys() {
			if !viewConfigurationKeys[k] {
				continue
			}
			v, _ := d.Get(k)
			c.Set(k, v)
		}
		input[i] = c
	}
	return svc.Create(ctx, input)
}

// UpdateViewConfigurations mirrors updateViewConfigurations(id, data): a
// configuration update replaces the configuration JSON (upsertWithReplace,
// missing keys reset to their defaults) instead of merging it. The result
// of the replace path is the upserted data (no columns read back).
func (s *SettingsModuleService) UpdateViewConfigurations(ctx context.Context, id string, data *dal.Record) (*dal.Record, error) {
	svc := s.Internal("ViewConfiguration")
	selector := dal.NewRecord()
	selector.Set("id", id)
	cfgRaw, _ := data.Get("configuration")
	if cfg, ok := cfgRaw.(*dal.Record); ok && cfg != nil {
		entities, err := svc.List(ctx, selector, utilsmodulessdk.FindConfig{})
		if err != nil {
			return nil, err
		}
		if len(entities) == 0 {
			return nil, nil
		}
		orDefault := func(key string, def any) any {
			if v, ok := cfg.Get(key); ok {
				return v
			}
			return def
		}
		orEmptyList := func(key string) any {
			if v, ok := cfg.Get(key); ok && v != nil {
				return v
			}
			return []any{}
		}
		rows := make([]*dal.Record, len(entities))
		for i, e := range entities {
			eid, _ := e.Get("id")
			row := dal.NewRecord()
			row.Set("id", eid)
			for _, k := range data.Keys() {
				v, _ := data.Get(k)
				row.Set(k, v)
			}
			normalized := dal.NewRecord()
			normalized.Set("visible_columns", orEmptyList("visible_columns"))
			normalized.Set("column_order", orEmptyList("column_order"))
			normalized.Set("column_widths", orDefault("column_widths", dal.NewRecord()))
			normalized.Set("filters", orDefault("filters", dal.NewRecord()))
			normalized.Set("sorting", orDefault("sorting", nil))
			normalized.Set("search", orDefault("search", ""))
			row.Set("configuration", normalized)
			rows[i] = row
		}
		out, err := svc.UpsertWithReplace(ctx, rows, nil)
		if err != nil || len(out) == 0 {
			return nil, err
		}
		return out[0], nil
	}
	in := dal.NewRecord()
	in.Set("selector", selector)
	in.Set("data", data)
	updated, err := svc.Update(ctx, in)
	if err != nil {
		return nil, err
	}
	list, _ := updated.([]*dal.Record)
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

// RetrieveViewConfiguration mirrors retrieveViewConfiguration(id, {}).
func (s *SettingsModuleService) RetrieveViewConfiguration(ctx context.Context, id string) (*dal.Record, error) {
	return s.Internal("ViewConfiguration").Retrieve(ctx, id, utilsmodulessdk.FindConfig{})
}

// DeleteViewConfigurations mirrors deleteViewConfigurations (hard delete).
func (s *SettingsModuleService) DeleteViewConfigurations(ctx context.Context, ids []any) error {
	_, err := s.Internal("ViewConfiguration").Delete(ctx, ids)
	return err
}

// GetUserPreference mirrors getUserPreference (nil when unset).
func (s *SettingsModuleService) GetUserPreference(ctx context.Context, userID, key string) (*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("user_id", userID)
	f.Set("key", key)
	take := 1
	rows, err := s.Internal("UserPreference").List(ctx, f, utilsmodulessdk.FindConfig{Take: &take})
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0], nil
}

// SetUserPreference mirrors setUserPreference.
func (s *SettingsModuleService) SetUserPreference(ctx context.Context, userID, key string, value any) error {
	f := dal.NewRecord()
	f.Set("user_id", userID)
	f.Set("key", key)
	take := 1
	existing, err := s.Internal("UserPreference").List(ctx, f, utilsmodulessdk.FindConfig{Take: &take, Select: []string{"id"}})
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		id, _ := existing[0].Get("id")
		u := dal.NewRecord()
		u.Set("id", id)
		u.Set("value", value)
		_, err = s.Internal("UserPreference").Update(ctx, []any{u})
		return err
	}
	c := dal.NewRecord()
	c.Set("user_id", userID)
	c.Set("key", key)
	c.Set("value", value)
	_, err = s.Internal("UserPreference").Create(ctx, []*dal.Record{c})
	return err
}

func activeViewKey(entity string) string { return "active_view." + entity }

// GetActiveViewConfiguration mirrors getActiveViewConfiguration: the
// preferred view (when it still exists), else the user's oldest view
// (unless the preference was explicitly cleared), else the system default.
func (s *SettingsModuleService) GetActiveViewConfiguration(ctx context.Context, entity, userID string) (*dal.Record, error) {
	pref, err := s.GetUserPreference(ctx, userID, activeViewKey(entity))
	if err != nil {
		return nil, err
	}
	var prefID any
	hasPrefID := false
	if pref != nil {
		if v, _ := pref.Get("value"); v != nil {
			if r, ok := v.(*dal.Record); ok {
				prefID, hasPrefID = r.Get("viewConfigurationId")
			}
		}
	}
	if Truthy(prefID) {
		if id, ok := prefID.(string); ok {
			if vc, err := s.RetrieveViewConfiguration(ctx, id); err == nil {
				return vc, nil
			}
		}
	}
	explicitNull := hasPrefID && prefID == nil
	if pref == nil || !explicitNull {
		f := dal.NewRecord()
		f.Set("entity", entity)
		f.Set("user_id", userID)
		take := 1
		rows, err := s.Internal("ViewConfiguration").List(ctx, f, utilsmodulessdk.FindConfig{Take: &take, Order: []dal.OrderBy{{Field: "created_at", Direction: "ASC"}}})
		if err != nil {
			return nil, err
		}
		if len(rows) > 0 {
			return rows[0], nil
		}
	}
	return s.GetSystemDefaultViewConfiguration(ctx, entity)
}

// GetSystemDefaultViewConfiguration mirrors getSystemDefaultViewConfiguration.
func (s *SettingsModuleService) GetSystemDefaultViewConfiguration(ctx context.Context, entity string) (*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("entity", entity)
	f.Set("is_system_default", true)
	rows, err := s.Internal("ViewConfiguration").List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0], nil
}

// SetActiveViewConfiguration mirrors setActiveViewConfiguration.
func (s *SettingsModuleService) SetActiveViewConfiguration(ctx context.Context, entity, userID, viewConfigurationID string) error {
	vc, err := s.RetrieveViewConfiguration(ctx, viewConfigurationID)
	if err != nil {
		return err
	}
	if e, _ := vc.Get("entity"); e != entity {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA,
			fmt.Sprintf("View configuration %s is not for entity %s", viewConfigurationID, entity))
	}
	if owner, _ := vc.Get("user_id"); Truthy(owner) && owner != userID {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED,
			fmt.Sprintf("User %s does not have access to view configuration %s", userID, viewConfigurationID))
	}
	v := dal.NewRecord()
	v.Set("viewConfigurationId", viewConfigurationID)
	return s.SetUserPreference(ctx, userID, activeViewKey(entity), v)
}

// ClearActiveViewConfiguration mirrors clearActiveViewConfiguration.
func (s *SettingsModuleService) ClearActiveViewConfiguration(ctx context.Context, entity, userID string) error {
	v := dal.NewRecord()
	v.Set("viewConfigurationId", nil)
	return s.SetUserPreference(ctx, userID, activeViewKey(entity), v)
}

// ListPropertyLabels mirrors listPropertyLabels(filters).
func (s *SettingsModuleService) ListPropertyLabels(ctx context.Context, filters *dal.Record) ([]*dal.Record, error) {
	return s.Internal("PropertyLabel").List(ctx, filters, utilsmodulessdk.FindConfig{})
}

// CreatePropertyLabels mirrors createPropertyLabels(array).
func (s *SettingsModuleService) CreatePropertyLabels(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	return s.Internal("PropertyLabel").Create(ctx, data)
}

// UpdatePropertyLabels mirrors updatePropertyLabels(array).
func (s *SettingsModuleService) UpdatePropertyLabels(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	in := make([]any, len(data))
	for i, d := range data {
		in[i] = d
	}
	out, err := s.Internal("PropertyLabel").Update(ctx, in)
	if err != nil {
		return nil, err
	}
	list, _ := out.([]*dal.Record)
	return list, nil
}

// DeletePropertyLabels mirrors deletePropertyLabels (hard delete).
func (s *SettingsModuleService) DeletePropertyLabels(ctx context.Context, ids []any) error {
	_, err := s.Internal("PropertyLabel").Delete(ctx, ids)
	return err
}

// UpsertPropertyLabels mirrors upsertPropertyLabels.
func (s *SettingsModuleService) UpsertPropertyLabels(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	// no @EmitEvents upstream
	return s.Internal("PropertyLabel").Upsert(dal.WithoutEmit(ctx), data)
}
