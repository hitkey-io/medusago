// Package workflows mirrors @medusajs/core-flows/src/settings/workflows.
package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/settings/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// SetLayoutConfigurationWorkflow mirrors setLayoutConfigurationWorkflow: the
// saved scope becomes the user's active layout scope for the zone.
var SetLayoutConfigurationWorkflow = composer.CreateWorkflow("set-layout-configuration",
	func(e *composer.Execution, in steps.SetLayoutConfigurationStepInput) (*dal.Record, error) {
		cfg, err := steps.SetLayoutConfigurationStep.Run(e, in)
		if err != nil {
			return nil, err
		}
		scope := "personal"
		if in.IsDefault {
			scope = "default"
		}
		if _, err := steps.SetActiveLayoutScopeStep.Run(e, steps.SetActiveLayoutScopeStepInput{Zone: in.Zone, UserID: in.UserID, Scope: scope}); err != nil {
			return nil, err
		}
		return cfg, nil
	})

// ClearLayoutConfigurationWorkflow mirrors clearLayoutConfigurationWorkflow
// (the active scope is reset: the input has no scope).
var ClearLayoutConfigurationWorkflow = composer.CreateWorkflow("clear-layout-configuration",
	func(e *composer.Execution, in steps.ClearLayoutConfigurationStepInput) (any, error) {
		if _, err := steps.ClearLayoutConfigurationStep.Run(e, in); err != nil {
			return nil, err
		}
		_, err := steps.SetActiveLayoutScopeStep.Run(e, steps.SetActiveLayoutScopeStepInput{Zone: in.Zone, UserID: in.UserID})
		return nil, err
	})
