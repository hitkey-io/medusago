package services

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// The layout configuration methods of SettingsModuleService.

func (s *SettingsModuleService) firstLayout(ctx context.Context, filters *dal.Record, sel []string) (*dal.Record, error) {
	take := 1
	rows, err := s.Internal("LayoutConfiguration").List(ctx, filters, utilsmodulessdk.FindConfig{Take: &take, Select: sel})
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0], nil
}

// GetSystemDefaultLayoutConfiguration mirrors getSystemDefaultLayoutConfiguration
// (nil when the zone has none).
func (s *SettingsModuleService) GetSystemDefaultLayoutConfiguration(ctx context.Context, zone string) (*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("zone", zone)
	f.Set("is_system_default", true)
	return s.firstLayout(ctx, f, nil)
}

// UpsertLayoutConfiguration mirrors upsertLayoutConfiguration_: the layout of
// the zone and user (nil for the system default) gets the configuration
// ({ widgets }) replaced, not merged; it is created when missing.
func (s *SettingsModuleService) UpsertLayoutConfiguration(ctx context.Context, zone string, userID any, isSystemDefault bool, configuration *dal.Record) (*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("zone", zone)
	f.Set("user_id", userID)
	existing, err := s.firstLayout(ctx, f, []string{"id"})
	if err != nil {
		return nil, err
	}
	widgets := any(dal.NewRecord())
	if configuration != nil {
		if w, ok := configuration.Get("widgets"); ok && w != nil {
			widgets = w
		}
	}
	normalized := dal.NewRecord()
	normalized.Set("widgets", widgets)
	payload := dal.NewRecord()
	if existing != nil {
		id, _ := existing.Get("id")
		payload.Set("id", id)
		payload.Set("configuration", normalized)
	} else {
		payload.Set("zone", zone)
		payload.Set("user_id", userID)
		payload.Set("is_system_default", isSystemDefault)
		payload.Set("configuration", normalized)
	}
	out, err := s.Internal("LayoutConfiguration").UpsertWithReplace(ctx, []*dal.Record{payload}, nil)
	if err != nil || len(out) == 0 {
		return nil, err
	}
	return out[0], nil
}

// SetLayoutConfiguration mirrors setLayoutConfiguration (the user's layout).
func (s *SettingsModuleService) SetLayoutConfiguration(ctx context.Context, zone, userID string, configuration *dal.Record) (*dal.Record, error) {
	return s.UpsertLayoutConfiguration(ctx, zone, userID, false, configuration)
}

// SetSystemDefaultLayoutConfiguration mirrors setSystemDefaultLayoutConfiguration.
func (s *SettingsModuleService) SetSystemDefaultLayoutConfiguration(ctx context.Context, zone string, configuration *dal.Record) (*dal.Record, error) {
	return s.UpsertLayoutConfiguration(ctx, zone, nil, true, configuration)
}

// ClearLayoutConfiguration mirrors clearLayoutConfiguration: the user's
// layouts of the zone are deleted.
func (s *SettingsModuleService) ClearLayoutConfiguration(ctx context.Context, zone, userID string) error {
	f := dal.NewRecord()
	f.Set("zone", zone)
	f.Set("user_id", userID)
	rows, err := s.Internal("LayoutConfiguration").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id"}})
	if err != nil || len(rows) == 0 {
		return err
	}
	ids := make([]any, len(rows))
	for i, r := range rows {
		ids[i], _ = r.Get("id")
	}
	_, err = s.Call(ctx, "deleteLayoutConfigurations", []any{ids})
	return err
}

func activeLayoutScopeKey(zone string) string { return "active_layout." + zone }

// GetActiveLayoutScope mirrors getActiveLayoutScope: "personal", "default"
// or nil.
func (s *SettingsModuleService) GetActiveLayoutScope(ctx context.Context, zone, userID string) (any, error) {
	pref, err := s.GetUserPreference(ctx, userID, activeLayoutScopeKey(zone))
	if err != nil || pref == nil {
		return nil, err
	}
	v, _ := pref.Get("value")
	r, _ := v.(*dal.Record)
	if r == nil {
		return nil, nil
	}
	scope, _ := r.Get("scope")
	if scope == "personal" || scope == "default" {
		return scope, nil
	}
	return nil, nil
}

// SetActiveLayoutScope mirrors setActiveLayoutScope (scope "personal",
// "default" or nil).
func (s *SettingsModuleService) SetActiveLayoutScope(ctx context.Context, zone, userID string, scope any) error {
	v := dal.NewRecord()
	v.Set("scope", scope)
	return s.SetUserPreference(ctx, userID, activeLayoutScopeKey(zone), v)
}
