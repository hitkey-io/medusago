// Package workflows mirrors @medusajs/core-flows/src/search/workflows.
package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/search/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// DeleteSearchIndexWorkflowID mirrors deleteSearchIndexWorkflowId.
const DeleteSearchIndexWorkflowID = "delete-search-index"

// DeleteSearchIndexWorkflow mirrors deleteSearchIndexWorkflow: validate the
// index, drop it, then the searchIndexDeleted hook.
var DeleteSearchIndexWorkflow = composer.CreateWorkflow(DeleteSearchIndexWorkflowID,
	func(e *composer.Execution, in steps.SearchIndexesInput) (*dal.Record, error) {
		if _, err := steps.ValidateSearchIndexesExistStep.Run(e, in); err != nil {
			return nil, err
		}
		result, err := steps.DeleteSearchIndexStep.Run(e, in)
		if err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		if len(in.Index) > 0 {
			hookInput.Set("index", in.Index[0])
		}
		if err := composer.CreateHook(e, "searchIndexDeleted", hookInput); err != nil {
			return nil, err
		}
		return result, nil
	})

// ReindexSearchIndexesWorkflowID mirrors reindexSearchIndexesWorkflowId.
const ReindexSearchIndexesWorkflowID = "reindex-search-indexes"

// ReindexSearchIndexesWorkflow mirrors reindexSearchIndexesWorkflow: the
// reindex step runs in the background (async, backgroundExecution).
var ReindexSearchIndexesWorkflow = composer.CreateWorkflow(ReindexSearchIndexesWorkflowID,
	func(e *composer.Execution, in steps.SearchIndexesInput) (any, error) {
		if _, err := steps.ValidateSearchIndexesExistStep.Run(e, in); err != nil {
			return nil, err
		}
		result, err := steps.ReindexSearchIndexesStep.Config("", composer.StepOptions{Async: true, BackgroundExecution: true}).Run(e, in)
		if err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		if result != nil {
			v, _ := result.Get("job_id")
			hookInput.Set("job_id", v)
			v, _ = result.Get("indexes")
			hookInput.Set("indexes", v)
		}
		if err := composer.CreateHook(e, "searchIndexesReindexed", hookInput); err != nil {
			return nil, err
		}
		return nil, nil
	})
