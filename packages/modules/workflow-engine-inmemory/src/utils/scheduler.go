// Package utils mirrors @medusajs/workflow-engine-inmemory/src/utils: the
// scheduler and the expired executions cleaner of
// InMemoryDistributedTransactionStorage.
package utils

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	cronparser "github.com/hitkey-io/medusago/packages/core/utils/src/cron-parser"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ThirtyMinutes mirrors THIRTY_MINUTES_IN_MS.
const ThirtyMinutes = 30 * time.Minute

type scheduledJob struct {
	timer              *time.Timer
	expression         *cronparser.Expression
	interval           int64
	numberOfExecutions int
	config             composer.SchedulerOptions
}

// Scheduler mirrors the scheduler part of
// InMemoryDistributedTransactionStorage: one timer per scheduled workflow.
type Scheduler struct {
	mu        sync.Mutex
	scheduled map[string]*scheduledJob
	container composer.Container
	logger    *slog.Logger
	closed    bool
}

// NewScheduler builds the scheduler; scheduled workflows run in the
// container.
func NewScheduler(c composer.Container, logger *slog.Logger) *Scheduler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Scheduler{scheduled: map[string]*scheduledJob{}, container: c, logger: logger}
}

// calculateDelayFromExpression mirrors calculateDelayFromExpression.
func calculateDelayFromExpression(e *cronparser.Expression) time.Duration {
	next, err := e.Next()
	if err != nil {
		return time.Millisecond
	}
	delay := time.Until(next)
	if delay <= 0 {
		nn, err := e.Next()
		if err != nil {
			return time.Millisecond
		}
		return max(time.Millisecond, time.Until(nn))
	}
	return delay
}

func (j *scheduledJob) nextExecution() time.Duration {
	if j.expression != nil {
		return calculateDelayFromExpression(j.expression)
	}
	return time.Duration(j.interval) * time.Millisecond
}

// Schedule mirrors schedule(jobDefinition, schedulerOptions).
func (s *Scheduler) Schedule(jobID string, opts composer.SchedulerOptions) error {
	_ = s.Remove(jobID)
	job := &scheduledJob{config: opts}
	switch {
	case opts.Cron != "":
		e, err := cronparser.Parse(opts.Cron, cronparser.Options{})
		if err != nil {
			return err
		}
		job.expression = e
	case opts.Interval != 0:
		job.interval = opts.Interval
	default:
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_ARGUMENT, "Schedule cron or interval definition is required for scheduled jobs.")
	}
	delay := job.nextExecution()
	scheduledFor := time.Now().Add(delay)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	job.timer = time.AfterFunc(delay, func() { s.jobHandler(jobID, scheduledFor) })
	s.scheduled[jobID] = job
	return nil
}

// Remove mirrors remove(jobId).
func (s *Scheduler) Remove(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, ok := s.scheduled[jobID]; ok {
		job.timer.Stop()
		delete(s.scheduled, jobID)
	}
	return nil
}

// RemoveAll mirrors removeAll / the timers cleared at shutdown.
func (s *Scheduler) RemoveAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, job := range s.scheduled {
		job.timer.Stop()
		delete(s.scheduled, id)
	}
	s.closed = true
}

// jobHandler mirrors jobHandler: the workflow runs with { scheduledFor }
// and the next execution is scheduled, until numberOfExecutions.
func (s *Scheduler) jobHandler(jobID string, scheduledFor time.Time) {
	s.mu.Lock()
	job := s.scheduled[jobID]
	if job == nil || s.closed {
		s.mu.Unlock()
		return
	}
	if n := job.config.NumberOfExecutions; n != nil && *n <= job.numberOfExecutions {
		delete(s.scheduled, jobID)
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	next := job.nextExecution()
	nextScheduledFor := time.Now().Add(next)
	input := dal.NewRecord()
	input.Set("scheduledFor", scheduledFor.UTC().Format("2006-01-02T15:04:05.000Z"))
	_, _, err := composer.RunByID(context.Background(), s.container, jobID, input, composer.RunOptions{})
	var me *common.MedusaError
	if err != nil && errors.As(err, &me) && me.Type == common.MedusaErrorTypes.NOT_FOUND && !composer.IsRegisteredWorkflow(jobID) {
		s.logger.Warn(fmt.Sprintf("Tried to execute a scheduled workflow with ID %s that does not exist, removing it from the scheduler.", jobID))
		_ = s.Remove(jobID)
		return
	}
	if err != nil {
		// logOnError, throwOnError false
		s.logger.Error(fmt.Sprintf("%s: %v", jobID, err))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.scheduled[jobID] != job {
		return
	}
	job.numberOfExecutions++
	job.timer = time.AfterFunc(next, func() { s.jobHandler(jobID, nextScheduledFor) })
}

// ClearExpiredExecutions mirrors clearExpiredExecutions: finished runs kept
// past their retention time are deleted.
func ClearExpiredExecutions(ctx context.Context, db dal.Querier) error {
	_, err := db.Exec(ctx, `delete from "workflow_execution" where "retention_time" is not null
and "updated_at" <= CURRENT_TIMESTAMP - (INTERVAL '1 second' * "retention_time")
and "state" in ('done', 'failed', 'reverted') and "deleted_at" is null`)
	return dal.MapDBError(err)
}
