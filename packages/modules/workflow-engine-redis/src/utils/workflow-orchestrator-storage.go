// Package utils mirrors @medusajs/workflow-engine-redis/src/utils:
// RedisDistributedTransactionStorage.
package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/bullmq"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/ioredis"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// Job names (JobType) and constants of the storage.
const (
	JobTypeSchedule           = "schedule"
	JobTypeRetry              = "retry"
	JobTypeStepTimeout        = "step_timeout"
	JobTypeTransactionTimeout = "transaction_timeout"

	ThirtyMinutesInMs        = 1000 * 60 * 30
	RepeatableClearerJobID   = "clear-expired-executions"
	CleanerQueueName         = "workflows-cleaner"
	DistributedTransactionKP = "dtrx"
	// defaultTTL mirrors TransactionOrchestrator.DEFAULT_TTL (seconds): the
	// first checkpoint of a flow without async steps expires.
	defaultTTL = 30 * time.Second
)

// StorageOptions are the resolved loader registrations.
type StorageOptions struct {
	DB                   dal.Querier
	Client               *ioredis.Client
	WorkerClient         *ioredis.Client
	QueueName            string
	JobQueueName         string
	MainQueueOptions     map[string]any
	JobQueueOptions      map[string]any
	JobWorkerOptions     map[string]any
	CleanerQueueOptions  map[string]any
	CleanerWorkerOptions map[string]any
	Logger               *slog.Logger
	IsWorkerMode         bool
	Container            composer.Container
}

// RedisDistributedTransactionStorage mirrors
// RedisDistributedTransactionStorage: checkpoints of the running stored
// transactions under dtrx:<workflowId>:<transactionId>, the
// workflow_execution rows, the scheduled workflows as repeatable jobs of
// the job queue and the expired executions cleaner.
//
// The main queue (retries, step and transaction timeouts) belongs to the
// orchestrator's checkpoint resumption: Go runs retry and wait inline in the
// process that runs the workflow, so it adds no jobs there and does not
// consume it (a Go worker could not resume a Node.js transaction).
type RedisDistributedTransactionStorage struct {
	opts          StorageOptions
	queue         *bullmq.Queue
	jobQueue      *bullmq.Queue
	cleanerQueue  *bullmq.Queue
	jobWorker     *bullmq.Worker
	cleanerWorker *bullmq.Worker
	logger        *slog.Logger
}

// NewRedisDistributedTransactionStorage mirrors the constructor.
func NewRedisDistributedTransactionStorage(ctx context.Context, opts StorageOptions) *RedisDistributedTransactionStorage {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	s := &RedisDistributedTransactionStorage{opts: opts, logger: logger}
	s.queue = bullmq.NewQueue(ctx, opts.QueueName, opts.Client.Client, bullmq.QueueOptionsFromMap(opts.MainQueueOptions))
	if opts.IsWorkerMode {
		s.jobQueue = bullmq.NewQueue(ctx, opts.JobQueueName, opts.Client.Client, bullmq.QueueOptionsFromMap(opts.JobQueueOptions))
		s.cleanerQueue = bullmq.NewQueue(ctx, CleanerQueueName, opts.Client.Client, bullmq.QueueOptionsFromMap(opts.CleanerQueueOptions))
	}
	return s
}

// OnApplicationStart mirrors onApplicationStart: the repeatable jobs of the
// main queue are removed (legacy schedules), the job and cleaner workers
// run and the cleaner is scheduled every 30 minutes.
func (s *RedisDistributedTransactionStorage) OnApplicationStart(ctx context.Context) error {
	if err := removeAllRepeatableJobs(ctx, s.queue); err != nil {
		return err
	}
	if !s.opts.IsWorkerMode {
		return nil
	}
	jw := bullmq.WorkerOptionsFromMap(s.opts.JobWorkerOptions)
	jw.Prefix = s.jobQueue.Prefix()
	jw.Logger = s.logger
	s.jobWorker = bullmq.NewWorker(s.opts.JobQueueName, s.opts.WorkerClient.Client, s.processScheduledJob, jw)
	cw := bullmq.WorkerOptionsFromMap(s.opts.CleanerWorkerOptions)
	cw.Prefix = s.cleanerQueue.Prefix()
	cw.Logger = s.logger
	s.cleanerWorker = bullmq.NewWorker(CleanerQueueName, s.opts.WorkerClient.Client, func(ctx context.Context, _ *bullmq.Job) (any, error) {
		return nil, s.ClearExpiredExecutions(ctx)
	}, cw)
	s.jobWorker.Run(ctx)
	s.cleanerWorker.Run(ctx)
	_, err := s.cleanerQueue.Add(ctx, "cleaner", bullmq.NewObj(), bullmq.NewObj(
		"repeat", bullmq.NewObj("every", ThirtyMinutesInMs),
		"jobId", RepeatableClearerJobID,
		"removeOnComplete", true,
		"removeOnFail", true,
	))
	return err
}

// OnApplicationShutdown mirrors onApplicationPrepareShutdown: the workers
// finish their jobs.
func (s *RedisDistributedTransactionStorage) OnApplicationShutdown() {
	if s.jobWorker != nil {
		s.jobWorker.Close()
	}
	if s.cleanerWorker != nil {
		s.cleanerWorker.Close()
	}
}

func removeAllRepeatableJobs(ctx context.Context, q *bullmq.Queue) error {
	jobs, err := q.GetRepeatableJobs(ctx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		if _, err := q.RemoveRepeatableByKey(ctx, j.Key); err != nil {
			return err
		}
	}
	return nil
}

// processScheduledJob mirrors the job worker: executeScheduledJob(jobId,
// schedulerOptions, new Date(job.timestamp)).
func (s *RedisDistributedTransactionStorage) processScheduledJob(ctx context.Context, job *bullmq.Job) (any, error) {
	data, _ := job.Data.(*bullmq.Obj)
	jobID, _ := mustGet(data, "jobId").(string)
	s.logger.Debug(fmt.Sprintf("executing scheduled job %s from queue %s", jobID, s.opts.JobQueueName))
	scheduledFor := time.UnixMilli(job.Timestamp).UTC().Format("2006-01-02T15:04:05.000Z")
	input := dal.NewRecord()
	input.Set("scheduledFor", scheduledFor)
	_, _, err := composer.RunByID(ctx, s.opts.Container, jobID, input, composer.RunOptions{})
	if err != nil {
		var me *common.MedusaError
		if errors.As(err, &me) && me.Type == common.MedusaErrorTypes.NOT_FOUND && !composer.IsRegisteredWorkflow(jobID) {
			s.logger.Warn(fmt.Sprintf("Tried to execute a scheduled workflow with ID %s that does not exist, removing it from the scheduler.", jobID))
			return nil, s.Remove(jobID)
		}
		return nil, err
	}
	return nil, nil
}

func mustGet(o *bullmq.Obj, k string) any {
	v, _ := o.Get(k)
	return v
}

// Schedule mirrors schedule(jobDefinition, schedulerOptions): a repeatable
// "schedule" job keyed schedule_<jobId> in the job queue.
func (s *RedisDistributedTransactionStorage) Schedule(jobID string, opts composer.SchedulerOptions) error {
	if opts.Cron != "" && opts.Interval != 0 {
		return errors.New("Unable to register a job with both scheduler options interval and cron.")
	}
	if s.jobQueue == nil {
		return nil
	}
	var limit any
	if opts.NumberOfExecutions != nil {
		limit = *opts.NumberOfExecutions
	}
	repeat := bullmq.NewObj("limit", limit, "key", JobTypeSchedule+"_"+jobID)
	if opts.Cron != "" {
		repeat.Set("pattern", opts.Cron)
	} else {
		repeat.Set("every", opts.Interval)
	}
	_, err := s.jobQueue.Add(context.Background(), JobTypeSchedule,
		bullmq.NewObj("jobId", jobID, "schedulerOptions", bullmq.ObjFromMap(opts.Record())),
		bullmq.NewObj(
			"repeat", repeat,
			"removeOnComplete", bullmq.NewObj("age", 86400, "count", 1000),
			"removeOnFail", bullmq.NewObj("age", 604800, "count", 5000),
		))
	return err
}

// Remove mirrors remove(jobId).
func (s *RedisDistributedTransactionStorage) Remove(jobID string) error {
	if s.jobQueue == nil {
		return nil
	}
	_, err := s.jobQueue.RemoveRepeatableByKey(context.Background(), JobTypeSchedule+"_"+jobID)
	return err
}

// RemoveAll mirrors removeAll.
func (s *RedisDistributedTransactionStorage) RemoveAll(ctx context.Context) error {
	if s.jobQueue == nil {
		return nil
	}
	return removeAllRepeatableJobs(ctx, s.jobQueue)
}

// KeyName mirrors TransactionOrchestrator.getKeyName("dtrx", workflowId,
// transactionId).
func KeyName(workflowID, transactionID string) string {
	return strings.Join([]string{DistributedTransactionKP, workflowID, transactionID}, ":")
}

var finishedStates = map[string]bool{
	composer.TransactionStateDone: true, composer.TransactionStateFailed: true, composer.TransactionStateReverted: true,
}

// SaveExecution mirrors save(key, checkpoint): the checkpoint
// { errors, flow } is kept in Redis while the transaction runs (set only
// if absent for the first checkpoint of a transaction with a given id) and
// removed when it finishes; the workflow_execution row is upserted.
func (s *RedisDistributedTransactionStorage) SaveExecution(ctx context.Context, row composer.ExecutionRow) error {
	if s.opts.DB != nil {
		if err := upsertExecution(ctx, s.opts.DB, row); err != nil {
			return err
		}
	}
	return s.saveCheckpoint(ctx, row)
}

func (s *RedisDistributedTransactionStorage) saveCheckpoint(ctx context.Context, row composer.ExecutionRow) error {
	key := s.opts.Client.KeyPrefix + KeyName(row.WorkflowID, row.TransactionID)
	if finishedStates[row.State] {
		return s.opts.Client.Unlink(ctx, key).Err()
	}
	var errs any = []any{}
	if c, ok := row.Context.(map[string]any); ok {
		errs = c["errors"]
	}
	checkpoint := dal.NewRecord()
	checkpoint.Set("errors", errs)
	checkpoint.Set("flow", row.Execution)
	b, err := json.Marshal(checkpoint)
	if err != nil {
		return err
	}
	var ttl time.Duration
	if row.State == composer.TransactionStateNotStarted {
		if flow, ok := row.Execution.(map[string]any); ok && flow["hasAsyncSteps"] != true {
			ttl = defaultTTL
		}
	}
	setNX := row.State == composer.TransactionStateNotStarted && !strings.HasPrefix(row.TransactionID, "auto-")
	if setNX {
		ok, err := s.opts.Client.SetNX(ctx, key, string(b), ttl).Result()
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("Transaction already started for transactionId: %s", row.TransactionID)
		}
		return nil
	}
	return s.opts.Client.Set(ctx, key, string(b), ttl).Err()
}

// DeleteExecution mirrors deleteFromDb + the checkpoint removal of a
// finished transaction.
func (s *RedisDistributedTransactionStorage) DeleteExecution(ctx context.Context, workflowID, transactionID, runID string) error {
	if s.opts.DB != nil {
		if _, err := s.opts.DB.Exec(ctx, `delete from "workflow_execution" where "run_id" = $1`, runID); err != nil {
			return dal.MapDBError(err)
		}
	}
	return s.opts.Client.Unlink(ctx, s.opts.Client.KeyPrefix+KeyName(workflowID, transactionID)).Err()
}

// upsertExecution mirrors workflowExecutionService.upsert([{ workflow_id,
// transaction_id, run_id, execution, context, state, retention_time }]).
func upsertExecution(ctx context.Context, db dal.Querier, row composer.ExecutionRow) error {
	execution, err := json.Marshal(row.Execution)
	if err != nil {
		return err
	}
	context_, err := json.Marshal(row.Context)
	if err != nil {
		return err
	}
	var retention any
	if row.RetentionTime != nil {
		retention = *row.RetentionTime
	}
	now := dal.Now()
	_, err = db.Exec(ctx, `insert into "workflow_execution" ("id", "workflow_id", "transaction_id", "run_id", "execution", "context", "state", "retention_time", "created_at", "updated_at")
values ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7, $8, $9, $9)
on conflict ("workflow_id", "transaction_id", "run_id") do update set "execution" = excluded."execution", "context" = excluded."context",
  "state" = excluded."state", "retention_time" = excluded."retention_time", "updated_at" = excluded."updated_at"`,
		common.GenerateEntityID("", "wf_exec"), row.WorkflowID, row.TransactionID, row.RunID, string(execution), string(context_), row.State, retention, now)
	return dal.MapDBError(err)
}

// ClearExpiredExecutions mirrors clearExpiredExecutions.
func (s *RedisDistributedTransactionStorage) ClearExpiredExecutions(ctx context.Context) error {
	_, err := s.opts.DB.Exec(ctx, `delete from "workflow_execution" where "retention_time" is not null
and "updated_at" <= CURRENT_TIMESTAMP - (INTERVAL '1 second' * "retention_time")
and "state" in ('done', 'failed', 'reverted') and "deleted_at" is null`)
	return dal.MapDBError(err)
}
