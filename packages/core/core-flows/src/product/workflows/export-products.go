package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	notificationsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/notification/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ExportProductsWorkflowID mirrors exportProductsWorkflowId.
const ExportProductsWorkflowID = "export-products"

// ExportProductsWorkflow mirrors exportProductsWorkflow.
var ExportProductsWorkflow = composer.CreateWorkflow(ExportProductsWorkflowID,
	func(e *composer.Execution, in steps.ExportProductsDTO) (any, error) {
		file, err := steps.ExportProductsStep.Config("", composer.StepOptions{Async: true, BackgroundExecution: true}).Run(e, in)
		if err != nil {
			return nil, err
		}
		failure := []*dal.Record{commonsteps.FeedNotification("", commonsteps.ExportFeedData("Product export", "Failed to export products, please try again later.", nil))}
		if _, err := notificationsteps.NotifyOnFailureStep.Run(e, failure); err != nil {
			return nil, err
		}
		filters := dal.NewRecord()
		filters.Set("id", file.ID)
		isList := false
		details, err := commonsteps.UseQueryGraphStep.Run(e, commonsteps.UseQueryGraphStepInput{GraphInput: query.GraphInput{Entity: "file", Fields: []string{"id", "url"}, Filters: filters}, IsList: &isList})
		if err != nil {
			return nil, err
		}
		var url any
		if r, ok := details.Data.(*dal.Record); ok {
			url, _ = r.Get("url")
		}
		notification := commonsteps.FeedNotification("", commonsteps.ExportFeedData("Product export", "Product export completed successfully!", commonsteps.ExportedFile(file.Filename, url, "text/csv")))
		_, err = notificationsteps.SendNotificationsStep.Run(e, []*dal.Record{notification})
		return nil, err
	})
