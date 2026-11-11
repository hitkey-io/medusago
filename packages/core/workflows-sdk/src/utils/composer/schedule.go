package composer

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// SchedulerOptions mirrors SchedulerOptions: a cron expression or an
// interval (milliseconds), the concurrency ("forbid") and an optional
// number of executions.
type SchedulerOptions struct {
	Cron               string
	Interval           int64
	Concurrency        string
	NumberOfExecutions *int
}

// Record is the JSON form ({ concurrency, cron | interval,
// numberOfExecutions }) stored with the scheduled job.
func (o SchedulerOptions) Record() *dal.Record {
	r := dal.NewRecord()
	if o.Concurrency != "" {
		r.Set("concurrency", o.Concurrency)
	}
	if o.Cron != "" {
		r.Set("cron", o.Cron)
	} else {
		r.Set("interval", o.Interval)
	}
	if o.NumberOfExecutions != nil {
		r.Set("numberOfExecutions", *o.NumberOfExecutions)
	}
	return r
}

// WorkflowScheduler mirrors IDistributedSchedulerStorage, implemented by
// the workflow engine module.
type WorkflowScheduler interface {
	Schedule(jobID string, opts SchedulerOptions) error
	Remove(jobID string) error
}

var (
	scheduleMu sync.Mutex
	scheduled  = map[string]SchedulerOptions{}
	// scheduledOrder keeps the registration order
	scheduledOrder []string
	scheduler      WorkflowScheduler
)

// normalizeSchedule mirrors WorkflowScheduler.scheduleWorkflow:
// { concurrency: "forbid", ...schedule }.
func normalizeSchedule(o SchedulerOptions) SchedulerOptions {
	if o.Concurrency == "" {
		o.Concurrency = "forbid"
	}
	return o
}

// registerScheduled mirrors WorkflowManager.register for a workflow with a
// schedule: it is scheduled now when a workflow engine is running, else
// when one starts.
func registerScheduled(id string, opts SchedulerOptions) {
	opts = normalizeSchedule(opts)
	scheduleMu.Lock()
	if _, ok := scheduled[id]; !ok {
		scheduledOrder = append(scheduledOrder, id)
	}
	scheduled[id] = opts
	s := scheduler
	scheduleMu.Unlock()
	if s != nil {
		if err := s.Schedule(id, opts); err != nil {
			slog.Error(fmt.Sprintf("failed to schedule workflow %s: %v", id, err))
		}
	}
}

// StartScheduler makes the workflow engine module the scheduler and
// schedules the workflows created with a schedule.
func StartScheduler(s WorkflowScheduler) error {
	scheduleMu.Lock()
	scheduler = s
	ids := append([]string(nil), scheduledOrder...)
	opts := make([]SchedulerOptions, len(ids))
	for i, id := range ids {
		opts[i] = scheduled[id]
	}
	scheduleMu.Unlock()
	for i, id := range ids {
		if err := s.Schedule(id, opts[i]); err != nil {
			return err
		}
	}
	return nil
}

// StopScheduler forgets the scheduler (the engine shuts down).
func StopScheduler(s WorkflowScheduler) {
	scheduleMu.Lock()
	if scheduler == s {
		scheduler = nil
	}
	scheduleMu.Unlock()
}

// UnregisterScheduled mirrors WorkflowManager.unregister of a scheduled
// workflow (clearWorkflow).
func UnregisterScheduled(id string) {
	scheduleMu.Lock()
	delete(scheduled, id)
	for i, v := range scheduledOrder {
		if v == id {
			scheduledOrder = append(scheduledOrder[:i:i], scheduledOrder[i+1:]...)
			break
		}
	}
	s := scheduler
	scheduleMu.Unlock()
	if s != nil {
		_ = s.Remove(id)
	}
}
