package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/store/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// DeleteStoresInput mirrors DeleteStoresWorkflowInput.
type DeleteStoresInput struct {
	IDs []any `json:"ids"`
}

// DeleteStoresWorkflow mirrors deleteStoresWorkflow: the stores are soft
// deleted and their links removed. No route runs it.
var DeleteStoresWorkflow = composer.CreateWorkflow("delete-stores",
	func(e *composer.Execution, input DeleteStoresInput) (any, error) {
		if _, err := steps.DeleteStoresStep.Run(e, input.IDs); err != nil {
			return nil, err
		}
		keys := dal.NewRecord()
		keys.Set("store_id", input.IDs)
		link := dal.NewRecord()
		link.Set(utilsmodulessdk.STORE, keys)
		_, err := commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{link})
		return nil, err
	})
