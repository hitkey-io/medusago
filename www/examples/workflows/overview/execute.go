package overview

import (
	"context"
	"log/slog"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	frameworkjobs "github.com/hitkey-io/medusago/packages/core/framework/src/jobs"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// region: api-route
// GET executes the workflow in an API route.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	result, err := HelloWorldWorkflow.Run(req.Ctx(), req.Scope, WorkflowInput{Name: "John"})
	if err != nil {
		return err
	}

	return res.JSON(result)
}

// endregion: api-route

// region: subscriber
// orderPlacedHandler executes the workflow in the subscriber of an event.
func orderPlacedHandler(container composer.Container) eventbus.Subscriber {
	return func(ctx context.Context, event eventbus.Event) error {
		result, err := HelloWorldWorkflow.Run(ctx, container, WorkflowInput{Name: "John"})
		if err != nil {
			return err
		}

		_ = result.Message
		return nil
	}
}

// endregion: subscriber

// region: job
// RegisterJob registers a scheduled job that executes the workflow once a
// day.
func RegisterJob() error {
	_, err := frameworkjobs.Register(
		&frameworkjobs.CronJobConfig{Name: "run-once-a-day", Schedule: "0 0 * * *"},
		func(ctx context.Context, container composer.Container, jc frameworkjobs.ScheduledJobContext) (any, error) {
			result, err := HelloWorldWorkflow.Run(ctx, container, WorkflowInput{Name: "John"})
			if err != nil {
				return nil, err
			}

			slog.Info(result.Message)
			return nil, nil
		},
	)
	return err
}

// endregion: job
