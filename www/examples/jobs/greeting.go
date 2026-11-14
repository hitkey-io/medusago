// Package jobs holds the scheduled jobs of the scheduled jobs chapter of the
// Learn documentation. In a project they live under src/jobs.
package jobs

import (
	"context"
	"log/slog"

	frameworkjobs "github.com/hitkey-io/medusago/packages/core/framework/src/jobs"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
)

// region: greeting
// greetingJob is the function the job runs. It receives the container, and
// a context with the time the run was scheduled for.
func greetingJob(ctx context.Context, container composer.Container, jc frameworkjobs.ScheduledJobContext) (any, error) {
	logger, _ := container.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)

	logger.Info("Greeting!", "scheduled_for", jc.ScheduledFor)
	return nil, nil
}

// greetingConfig holds the name of the job, which must be unique, and its
// schedule, a cron expression.
var greetingConfig = &frameworkjobs.CronJobConfig{
	Name:     "greeting-every-minute",
	Schedule: "* * * * *",
}

// endregion: greeting

// region: register
// RegisterJobs registers the jobs of the project. Pass it to
// cli.Options.OnLoad.
func RegisterJobs(ctx context.Context, app *loaders.MedusaApp) error {
	if _, err := frameworkjobs.Register(greetingConfig, greetingJob); err != nil {
		return err
	}
	if _, err := frameworkjobs.Register(syncProductsConfig, syncProductsJob); err != nil {
		return err
	}

	return nil
}

// endregion: register
