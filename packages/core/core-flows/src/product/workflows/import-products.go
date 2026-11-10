package workflows

import (
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	notificationsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/notification/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ImportProductsInput mirrors ImportProductsDTO.
type ImportProductsInput struct {
	Filename    string
	FileContent string
}

// ImportProductsWorkflowID mirrors importProductsWorkflowId.
const ImportProductsWorkflowID = "import-products"

// ImportProductsWorkflow mirrors importProductsWorkflow (the legacy import
// of a file uploaded with the request).
var ImportProductsWorkflow = composer.CreateWorkflow(ImportProductsWorkflowID,
	func(e *composer.Execution, in ImportProductsInput) (steps.ImportSummary, error) {
		batch, err := steps.NormalizeCsvStep.Run(e, in.FileContent)
		if err != nil {
			return steps.ImportSummary{}, err
		}
		summary := steps.ImportSummary{ToCreate: len(batch.Create), ToUpdate: len(batch.Update)}
		composer.SetPartialResult(e, summary)
		if _, err := steps.WaitConfirmationProductImportStep.Run(e, nil); err != nil {
			return steps.ImportSummary{}, err
		}
		failure := []*dal.Record{commonsteps.FeedNotification("", commonsteps.ExportFeedData("Product import", fmt.Sprintf("Failed to import products from file %s", in.Filename), nil))}
		if _, err := notificationsteps.NotifyOnFailureStep.Run(e, failure); err != nil {
			return steps.ImportSummary{}, err
		}
		if _, err := BatchProductsWorkflow.RunAsStep(e, BatchProductsInput{Create: batch.Create, Update: batch.Update}); err != nil {
			return steps.ImportSummary{}, err
		}
		done := []*dal.Record{commonsteps.FeedNotification("", commonsteps.ExportFeedData("Product import", fmt.Sprintf("Product import of file %s completed successfully!", in.Filename), nil))}
		if _, err := notificationsteps.SendNotificationsStep.Run(e, done); err != nil {
			return steps.ImportSummary{}, err
		}
		return summary, nil
	})
