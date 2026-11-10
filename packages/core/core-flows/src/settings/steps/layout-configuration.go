// Package steps mirrors @medusajs/core-flows/src/settings/steps.
package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// The settings module service methods the layout steps use
// (SettingsModuleService: setLayoutConfiguration, setUserPreference, ...),
// on the generic settings module service.

func settings(c composer.Container) *utilsmodulessdk.MedusaService {
	s, _ := c.Resolve(utilsmodulessdk.SETTINGS).(*utilsmodulessdk.MedusaService)
	return s
}

func firstRow(ctx context.Context, svc *utilsmodulessdk.MedusaService, model string, filters *dal.Record, sel []string) (*dal.Record, error) {
	take := 1
	rows, err := svc.Internal(model).List(ctx, filters, utilsmodulessdk.FindConfig{Take: &take, Select: sel})
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0], nil
}

func layoutFilter(zone string, userID any) *dal.Record {
	f := dal.NewRecord()
	f.Set("zone", zone)
	f.Set("user_id", userID)
	return f
}

// upsertLayoutConfiguration mirrors upsertLayoutConfiguration_: the
// configuration JSON is replaced ({ widgets }), not merged.
func upsertLayoutConfiguration(ctx context.Context, svc *utilsmodulessdk.MedusaService, zone string, userID any, isDefault bool, configuration *dal.Record) (*dal.Record, error) {
	existing, err := firstRow(ctx, svc, "LayoutConfiguration", layoutFilter(zone, userID), []string{"id"})
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
		payload.Set("is_system_default", isDefault)
		payload.Set("configuration", normalized)
	}
	out, err := svc.Internal("LayoutConfiguration").UpsertWithReplace(ctx, []*dal.Record{payload}, nil)
	if err != nil || len(out) == 0 {
		return nil, err
	}
	return out[0], nil
}

// setUserPreference mirrors setUserPreference.
func setUserPreference(ctx context.Context, svc *utilsmodulessdk.MedusaService, userID, key string, value any) error {
	f := dal.NewRecord()
	f.Set("user_id", userID)
	f.Set("key", key)
	existing, err := firstRow(ctx, svc, "UserPreference", f, []string{"id"})
	if err != nil {
		return err
	}
	if existing != nil {
		id, _ := existing.Get("id")
		u := dal.NewRecord()
		u.Set("id", id)
		u.Set("value", value)
		_, err = svc.Internal("UserPreference").Update(ctx, []any{u})
		return err
	}
	c := dal.NewRecord()
	c.Set("user_id", userID)
	c.Set("key", key)
	c.Set("value", value)
	_, err = svc.Internal("UserPreference").Create(ctx, []*dal.Record{c})
	return err
}

// getActiveLayoutScope mirrors getActiveLayoutScope (nil when unset).
func getActiveLayoutScope(ctx context.Context, svc *utilsmodulessdk.MedusaService, zone, userID string) (any, error) {
	f := dal.NewRecord()
	f.Set("user_id", userID)
	f.Set("key", "active_layout."+zone)
	pref, err := firstRow(ctx, svc, "UserPreference", f, nil)
	if err != nil || pref == nil {
		return nil, err
	}
	v, _ := pref.Get("value")
	r, _ := v.(*dal.Record)
	if r == nil {
		return nil, nil
	}
	s, _ := r.Get("scope")
	if s == "personal" || s == "default" {
		return s, nil
	}
	return nil, nil
}

func setActiveLayoutScope(ctx context.Context, svc *utilsmodulessdk.MedusaService, zone, userID string, scope any) error {
	v := dal.NewRecord()
	v.Set("scope", scope)
	return setUserPreference(ctx, svc, userID, "active_layout."+zone, v)
}

// SetLayoutConfigurationStepInput mirrors SetLayoutConfigurationStepInput.
type SetLayoutConfigurationStepInput struct {
	Zone          string
	UserID        string
	IsDefault     bool
	Configuration *dal.Record
}

type layoutCompensation struct {
	zone, userID     string
	isDefault        bool
	previous         any
	createdID        any
	restoreAsDefault bool
}

// SetLayoutConfigurationStep mirrors setLayoutConfigurationStep.
var SetLayoutConfigurationStep = composer.CreateStep("set-layout-configuration",
	func(ctx composer.StepExecutionContext, in SetLayoutConfigurationStepInput) (composer.StepResponse[*dal.Record], error) {
		svc := settings(ctx.Container)
		var userFilter any = in.UserID
		if in.IsDefault {
			userFilter = nil
		}
		previous, err := firstRow(ctx.Context, svc, "LayoutConfiguration", layoutFilter(in.Zone, userFilter), nil)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		var owner any = in.UserID
		if in.IsDefault {
			owner = nil
		}
		result, err := upsertLayoutConfiguration(ctx.Context, svc, in.Zone, owner, in.IsDefault, in.Configuration)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		comp := layoutCompensation{zone: in.Zone, userID: in.UserID, isDefault: in.IsDefault}
		if previous != nil {
			comp.previous, _ = previous.Get("configuration")
		} else {
			comp.createdID, _ = result.Get("id")
		}
		return composer.NewStepResponse(result, any(comp)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		comp, ok := data.(layoutCompensation)
		if !ok {
			return nil
		}
		svc := settings(ctx.Container)
		if prev, _ := comp.previous.(*dal.Record); prev != nil {
			var owner any = comp.userID
			if comp.isDefault {
				owner = nil
			}
			_, err := upsertLayoutConfiguration(ctx.Context, svc, comp.zone, owner, comp.isDefault, prev)
			return err
		}
		if comp.createdID != nil {
			_, err := svc.Internal("LayoutConfiguration").Delete(ctx.Context, []any{comp.createdID})
			return err
		}
		return nil
	})

// ClearLayoutConfigurationStepInput mirrors ClearLayoutConfigurationStepInput.
type ClearLayoutConfigurationStepInput struct {
	Zone   string
	UserID string
}

// ClearLayoutConfigurationStep mirrors clearLayoutConfigurationStep.
var ClearLayoutConfigurationStep = composer.CreateStep("clear-layout-configuration",
	func(ctx composer.StepExecutionContext, in ClearLayoutConfigurationStepInput) (composer.StepResponse[any], error) {
		svc := settings(ctx.Container)
		previous, err := firstRow(ctx.Context, svc, "LayoutConfiguration", layoutFilter(in.Zone, in.UserID), nil)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		rows, err := svc.Internal("LayoutConfiguration").List(ctx.Context, layoutFilter(in.Zone, in.UserID), utilsmodulessdk.FindConfig{Select: []string{"id"}})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		if len(rows) > 0 {
			ids := make([]any, len(rows))
			for i, r := range rows {
				ids[i], _ = r.Get("id")
			}
			if _, err := svc.Internal("LayoutConfiguration").Delete(ctx.Context, ids); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		comp := layoutCompensation{zone: in.Zone, userID: in.UserID}
		if previous != nil {
			comp.previous, _ = previous.Get("configuration")
		}
		return composer.NewStepResponse[any](nil, any(comp)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		comp, ok := data.(layoutCompensation)
		prev, _ := comp.previous.(*dal.Record)
		if !ok || prev == nil {
			return nil
		}
		_, err := upsertLayoutConfiguration(ctx.Context, settings(ctx.Container), comp.zone, comp.userID, false, prev)
		return err
	})

// SetActiveLayoutScopeStepInput mirrors SetActiveLayoutScopeStepInput (nil
// Scope = null).
type SetActiveLayoutScopeStepInput struct {
	Zone   string
	UserID string
	Scope  any
}

// SetActiveLayoutScopeStep mirrors setActiveLayoutScopeStep.
var SetActiveLayoutScopeStep = composer.CreateStep("set-active-layout-scope",
	func(ctx composer.StepExecutionContext, in SetActiveLayoutScopeStepInput) (composer.StepResponse[any], error) {
		svc := settings(ctx.Container)
		previous, err := getActiveLayoutScope(ctx.Context, svc, in.Zone, in.UserID)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := setActiveLayoutScope(ctx.Context, svc, in.Zone, in.UserID, in.Scope); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse(in.Scope, any(SetActiveLayoutScopeStepInput{Zone: in.Zone, UserID: in.UserID, Scope: previous})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, ok := data.(SetActiveLayoutScopeStepInput)
		if !ok {
			return nil
		}
		return setActiveLayoutScope(ctx.Context, settings(ctx.Container), prev.Zone, prev.UserID, prev.Scope)
	})
