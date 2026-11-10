package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer/steps"
	notificationsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/notification/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ExportCustomersDTO mirrors ExportCustomersDTO.
type ExportCustomersDTO struct {
	Select    []string
	Filter    *dal.Record
	BatchSize any
	Format    string
	// To is the notification recipient (nil = "").
	To any
}

// ExportCustomersWorkflowID mirrors exportCustomersWorkflowId.
const ExportCustomersWorkflowID = "export-customers"

// ExportCustomersWorkflow mirrors exportCustomersWorkflow (hook
// customerDataExport).
var ExportCustomersWorkflow = composer.CreateWorkflow(ExportCustomersWorkflowID,
	func(e *composer.Execution, in ExportCustomersDTO) (steps.ExportCustomersStepOutput, error) {
		var zero steps.ExportCustomersStepOutput
		to := in.To
		if to == nil {
			to = ""
		}
		failure := []*dal.Record{commonsteps.FeedNotification(to, commonsteps.ExportFeedData("Customer export", "Failed to export customers, please try again later.", nil))}
		if _, err := notificationsteps.NotifyOnFailureStep.Run(e, failure); err != nil {
			return zero, err
		}
		ids, err := steps.CollectCustomerIdsToExportStep.Config("", composer.StepOptions{Async: true, BackgroundExecution: true}).
			Run(e, steps.CollectCustomerIdsToExportStepInput{Filter: in.Filter, BatchSize: in.BatchSize})
		if err != nil {
			return zero, err
		}
		hookInput := dal.NewRecord()
		idList := make([]any, len(ids))
		for i, id := range ids {
			idList[i] = id
		}
		hookInput.Set("customer_ids", idList)
		hookInput.Set("filter", in.Filter)
		extra, err := composer.CreateHookWithResult(e, "customerDataExport", hookInput)
		if err != nil {
			return zero, err
		}
		additional, _ := extra.(*dal.Record)
		selectFields := in.Select
		if selectFields == nil {
			selectFields = []string{}
		}
		file, err := steps.ExportCustomersStep.Run(e, steps.ExportCustomersStepInput{Select: selectFields, Format: in.Format, BatchSize: in.BatchSize, CustomerIDs: ids, AdditionalData: additional})
		if err != nil {
			return zero, err
		}
		filters := dal.NewRecord()
		filters.Set("id", file.ID)
		isList := false
		details, err := commonsteps.UseQueryGraphStep.Run(e, commonsteps.UseQueryGraphStepInput{GraphInput: query.GraphInput{Entity: "file", Fields: []string{"id", "url"}, Filters: filters}, IsList: &isList})
		if err != nil {
			return zero, err
		}
		var url any
		if r, ok := details.Data.(*dal.Record); ok {
			url, _ = r.Get("url")
		}
		mime := "application/json"
		if in.Format == "csv" {
			mime = "text/csv"
		}
		notification := commonsteps.FeedNotification(to, commonsteps.ExportFeedData("Customer export", "Customer export completed successfully!", commonsteps.ExportedFile(file.Filename, url, mime)))
		if _, err := notificationsteps.SendNotificationsStep.Run(e, []*dal.Record{notification}); err != nil {
			return zero, err
		}
		return file, nil
	})
