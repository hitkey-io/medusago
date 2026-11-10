package workflows

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/settings/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	settingsservices "github.com/hitkey-io/medusago/packages/modules/settings/src/services"
)

func str(r *dal.Record, key string) string {
	if r == nil {
		return ""
	}
	v, _ := r.Get(key)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func truthyKey(r *dal.Record, key string) bool {
	if r == nil {
		return false
	}
	v, _ := r.Get(key)
	return settingsservices.Truthy(v)
}

// CreateViewConfigurationWorkflow mirrors createViewConfigurationWorkflow
// (input: CreateViewConfigurationDTO & { set_active }). The created
// configuration becomes the user's active view when set_active and user_id
// are set.
var CreateViewConfigurationWorkflow = composer.CreateWorkflow("create-view-configuration",
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		viewConfig, err := steps.CreateViewConfigurationStep.Run(e, in)
		if err != nil {
			return nil, err
		}
		if truthyKey(in, "set_active") && truthyKey(in, "user_id") {
			if _, err := steps.SetActiveViewConfigurationStep.Run(e, steps.SetActiveViewConfigurationStepInput{
				ID: str(viewConfig, "id"), Entity: str(viewConfig, "entity"), UserID: str(in, "user_id"),
			}); err != nil {
				return nil, err
			}
		}
		return viewConfig, nil
	})

// UpdateViewConfigurationWorkflow mirrors updateViewConfigurationWorkflow
// (input: { id, set_active, ...UpdateViewConfigurationDTO }). The updated
// configuration becomes the user's active view when set_active is set and
// the step result carries a user_id (a configuration update returns the
// upserted data, without user_id).
var UpdateViewConfigurationWorkflow = composer.CreateWorkflow("update-view-configuration",
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		data := dal.NewRecord()
		for _, k := range in.Keys() {
			if k == "id" || k == "set_active" {
				continue
			}
			v, _ := in.Get(k)
			data.Set(k, v)
		}
		viewConfig, err := steps.UpdateViewConfigurationStep.Run(e, steps.UpdateViewConfigurationStepInput{ID: str(in, "id"), Data: data})
		if err != nil {
			return nil, err
		}
		if viewConfig == nil && truthyKey(in, "set_active") {
			return nil, fmt.Errorf("Cannot read properties of undefined (reading 'user_id')")
		}
		if truthyKey(in, "set_active") && truthyKey(viewConfig, "user_id") {
			if _, err := steps.SetActiveViewConfigurationStep.Run(e, steps.SetActiveViewConfigurationStepInput{
				ID: str(viewConfig, "id"), Entity: str(viewConfig, "entity"), UserID: str(viewConfig, "user_id"),
			}); err != nil {
				return nil, err
			}
		}
		return viewConfig, nil
	})
