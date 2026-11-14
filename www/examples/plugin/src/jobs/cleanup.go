// Package jobs holds the scheduled jobs of the plugin.
package jobs

import (
	"context"
	"log/slog"

	frameworkjobs "github.com/hitkey-io/medusago/packages/core/framework/src/jobs"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// region: body
// Register registers the scheduled jobs of the plugin.
func Register() {
	_, err := frameworkjobs.Register(
		&frameworkjobs.CronJobConfig{Name: "wishlist-cleanup", Schedule: "0 0 * * *"},
		func(ctx context.Context, container composer.Container, jc frameworkjobs.ScheduledJobContext) (any, error) {
			slog.Info("Cleaning up the wishlists", "scheduled_for", jc.ScheduledFor)
			return nil, nil
		},
	)
	if err != nil {
		slog.Error("Unable to register the job", "error", err)
	}
}

// endregion: body
