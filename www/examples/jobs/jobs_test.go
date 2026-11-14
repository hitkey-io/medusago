package jobs

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hitkey-io/medusago/packages/core/framework/src/container"
	frameworkjobs "github.com/hitkey-io/medusago/packages/core/framework/src/jobs"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/modules/workflow-engine-inmemory/src/utils"
)

func TestJobs(t *testing.T) {
	c := container.New()
	c.Register(common.ContainerRegistrationKeys.LOGGER, slog.Default())

	scheduler := utils.NewScheduler(c, nil)
	if err := composer.StartScheduler(scheduler); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		composer.StopScheduler(scheduler)
		scheduler.RemoveAll()
	})

	// the schedule of the example jobs is a valid cron expression
	if err := scheduler.Schedule("job-sync-products-job", composer.SchedulerOptions{Cron: syncProductsConfig.Schedule}); err != nil {
		t.Fatal(err)
	}
	if err := scheduler.Schedule("job-greeting-every-minute", composer.SchedulerOptions{Cron: greetingConfig.Schedule}); err != nil {
		t.Fatal(err)
	}

	// a job with options runs the number of times it asks for
	var runs atomic.Int32
	var scheduledFor atomic.Value
	cfg := &frameworkjobs.CronJobConfig{
		Name: "test-job",
		ScheduleOptions: &composer.SchedulerOptions{
			Interval:           20,
			NumberOfExecutions: new(2),
		},
	}
	if _, err := frameworkjobs.Register(cfg, func(ctx context.Context, c composer.Container, jc frameworkjobs.ScheduledJobContext) (any, error) {
		runs.Add(1)
		scheduledFor.Store(jc.ScheduledFor)
		return nil, nil
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { frameworkjobs.Unregister("test-job") })

	time.Sleep(400 * time.Millisecond)
	if got := runs.Load(); got != 2 {
		t.Fatalf("the job ran %d times, want 2", got)
	}
	if v, _ := scheduledFor.Load().(time.Time); v.IsZero() {
		t.Fatal("the job didn't receive the scheduled time")
	}

	// the handlers run
	if _, err := greetingJob(context.Background(), c, frameworkjobs.ScheduledJobContext{ScheduledFor: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if _, err := syncProductsJob(context.Background(), c, frameworkjobs.ScheduledJobContext{}); err != nil {
		t.Fatal(err)
	}
}
