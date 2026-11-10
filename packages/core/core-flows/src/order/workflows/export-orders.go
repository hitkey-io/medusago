package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	notificationsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/notification/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ExportOrdersDTO mirrors ExportOrdersDTO.
type ExportOrdersDTO struct {
	Select []string
	Filter *dal.Record
}

// ExportOrdersWorkflowID mirrors exportOrdersWorkflowId.
const ExportOrdersWorkflowID = "export-orders"

// ExportedFileNotification is the success notification of an export.
func ExportedFileNotification(ctx *composer.Execution, title, description, fileID, filename string) error {
	list := false
	v := dal.NewRecord()
	v.Set("id", fileID)
	details, err := commonsteps.UseRemoteQueryStep.Run(ctx, commonsteps.RemoteStepInput{EntryPoint: "file", Fields: []string{"id", "url"}, Variables: v, List: &list})
	if err != nil {
		return err
	}
	var url any
	if len(details.Data) > 0 {
		if r, ok := details.Data[0].(*dal.Record); ok {
			url, _ = r.Get("url")
		}
	}
	file := commonsteps.ExportedFile(filename, url, "text/csv")
	_, err = notificationsteps.SendNotificationsStep.Run(ctx, []*dal.Record{commonsteps.FeedNotification("", commonsteps.ExportFeedData(title, description, file))})
	return err
}

// ExportOrdersWorkflow mirrors exportOrdersWorkflow: the export runs in the
// background and the admin gets a feed notification.
var ExportOrdersWorkflow = composer.CreateWorkflow(ExportOrdersWorkflowID,
	func(e *composer.Execution, in ExportOrdersDTO) (any, error) {
		file, err := steps.ExportOrdersStep.Config("", composer.StepOptions{Async: true, BackgroundExecution: true}).
			Run(e, steps.ExportOrdersStepInput{Select: in.Select, Filter: in.Filter})
		if err != nil {
			return nil, err
		}
		failure := []*dal.Record{commonsteps.FeedNotification("", commonsteps.ExportFeedData("Order export", "Failed to export orders, please try again later.", nil))}
		if _, err := notificationsteps.NotifyOnFailureStep.Run(e, failure); err != nil {
			return nil, err
		}
		return nil, ExportedFileNotification(e, "Order export", "Order export completed successfully!", file.ID, file.Filename)
	})
