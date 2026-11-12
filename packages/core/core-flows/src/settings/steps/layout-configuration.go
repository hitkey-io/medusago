// Package steps mirrors @medusajs/core-flows/src/settings/steps.
package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	settingsservices "github.com/hitkey-io/medusago/packages/modules/settings/src/services"
)

// The settings module service methods the layout steps use
// (SettingsModuleService: setLayoutConfiguration, setUserPreference, ...),
// on the generated settings module service.

func settings(c composer.Container) *utilsmodulessdk.MedusaService {
	if s := settingsservices.Of(c.Resolve(utilsmodulessdk.SETTINGS)); s != nil {
		return s.MedusaService
	}
	return nil
}

// settingsModule resolves the settings module service.
func settingsModule(c composer.Container) *settingsservices.SettingsModuleService {
	return settingsservices.Of(c.Resolve(utilsmodulessdk.SETTINGS))
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

// upsertLayoutConfiguration mirrors upsertLayoutConfiguration_.
func upsertLayoutConfiguration(ctx context.Context, svc *utilsmodulessdk.MedusaService, zone string, userID any, isDefault bool, configuration *dal.Record) (*dal.Record, error) {
	return settingsservices.Of(svc).UpsertLayoutConfiguration(ctx, zone, userID, isDefault, configuration)
}

// setUserPreference mirrors setUserPreference.
func setUserPreference(ctx context.Context, svc *utilsmodulessdk.MedusaService, userID, key string, value any) error {
	return settingsservices.Of(svc).SetUserPreference(ctx, userID, key, value)
}

// getActiveLayoutScope mirrors getActiveLayoutScope (nil when unset).
func getActiveLayoutScope(ctx context.Context, svc *utilsmodulessdk.MedusaService, zone, userID string) (any, error) {
	return settingsservices.Of(svc).GetActiveLayoutScope(ctx, zone, userID)
}

func setActiveLayoutScope(ctx context.Context, svc *utilsmodulessdk.MedusaService, zone, userID string, scope any) error {
	return settingsservices.Of(svc).SetActiveLayoutScope(ctx, zone, userID, scope)
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
