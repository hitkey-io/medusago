// Package jobs mirrors @medusajs/framework/src/jobs: scheduled jobs. A job
// is registered (from a project's code, where upstream loads the files of
// src/jobs) as a workflow "job-<name>" with one step "<name>-as-step" and
// a schedule; the workflow engine module runs it on its scheduler.
package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ScheduledJobContext mirrors ScheduledJobContext.
type ScheduledJobContext struct {
	// ScheduledFor is the timestamp for which the job was scheduled.
	ScheduledFor time.Time
}

// ScheduledJobHandler mirrors ScheduledJobHandler.
type ScheduledJobHandler func(ctx context.Context, container composer.Container, jc ScheduledJobContext) (any, error)

// ScheduledJobWorkflowInput mirrors ScheduledJobWorkflowInput.
type ScheduledJobWorkflowInput struct {
	ScheduledFor string `json:"scheduledFor"`
}

// CronJobConfig mirrors the config export of a job file: a cron
// expression (Schedule) or scheduler options (ScheduleOptions).
type CronJobConfig struct {
	Name               string
	Schedule           string
	ScheduleOptions    *composer.SchedulerOptions
	NumberOfExecutions *int
}

// validateConfig mirrors validateConfig.
func validateConfig(config *CronJobConfig) error {
	if config == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_ARGUMENT, "Config is required for scheduled jobs.")
	}
	if config.Schedule == "" && config.ScheduleOptions == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_ARGUMENT, "Cron schedule definition is required for scheduled jobs.")
	}
	if config.Name == "" {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_ARGUMENT, "Job name is required for scheduled jobs.")
	}
	return nil
}

// Register mirrors JobLoader.register: the workflow job-<name> with the
// step <name>-as-step, scheduled with the config's schedule (a string is
// { cron, numberOfExecutions }).
func Register(config *CronJobConfig, handler ScheduledJobHandler) (*composer.Workflow[ScheduledJobWorkflowInput, any], error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	name := config.Name
	step := composer.CreateStep(name+"-as-step",
		func(ctx composer.StepExecutionContext, in ScheduledJobWorkflowInput) (composer.StepResponse[any], error) {
			jc := ScheduledJobContext{ScheduledFor: time.Now()}
			if in.ScheduledFor != "" {
				if t, ok := common.ParseJSDate(in.ScheduledFor); ok {
					jc.ScheduledFor = t
				}
			}
			res, err := handler(ctx.Context, ctx.Container, jc)
			if err != nil {
				slog.Error(fmt.Sprintf("Scheduled job %s failed with error: %s", name, err.Error()))
				return composer.StepResponse[any]{}, err
			}
			return composer.NewStepResponse[any](res, res), nil
		}, nil)
	schedule := config.ScheduleOptions
	if schedule == nil {
		schedule = &composer.SchedulerOptions{Cron: config.Schedule, NumberOfExecutions: config.NumberOfExecutions}
	}
	return composer.CreateWorkflowWithOptions("job-"+name, composer.WorkflowOptions{Schedule: schedule},
		func(e *composer.Execution, in ScheduledJobWorkflowInput) (any, error) {
			return step.Run(e, in)
		}), nil
}

// Unregister removes the job's workflow from the scheduler (WorkflowManager
// unregister of a scheduled workflow).
func Unregister(name string) {
	composer.UnregisterScheduled("job-" + name)
}
