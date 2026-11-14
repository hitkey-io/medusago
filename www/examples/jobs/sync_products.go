package jobs

import (
	"context"

	frameworkjobs "github.com/hitkey-io/medusago/packages/core/framework/src/jobs"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// SyncProductsToErpWorkflow stands for a workflow that syncs products to an
// ERP. Workflows are covered in their own chapter.
var SyncProductsToErpWorkflow = composer.CreateWorkflow("sync-products-to-erp",
	func(e *composer.Execution, input struct{}) (any, error) {
		return nil, nil
	})

// region: sync-products
// syncProductsJob runs the workflow once a day.
func syncProductsJob(ctx context.Context, container composer.Container, jc frameworkjobs.ScheduledJobContext) (any, error) {
	_, err := SyncProductsToErpWorkflow.Run(ctx, container, struct{}{})
	return nil, err
}

// syncProductsConfig schedules the job at midnight, every day.
var syncProductsConfig = &frameworkjobs.CronJobConfig{
	Name:     "sync-products-job",
	Schedule: "0 0 * * *",
}

// endregion: sync-products

// region: options
// reportConfig runs the job every ten seconds, five times: the job isn't
// scheduled again after that. new(5) is a pointer to the number 5.
var reportConfig = &frameworkjobs.CronJobConfig{
	Name: "report-job",
	ScheduleOptions: &composer.SchedulerOptions{
		// the interval is in milliseconds
		Interval:           10_000,
		NumberOfExecutions: new(5),
	},
}

// endregion: options
