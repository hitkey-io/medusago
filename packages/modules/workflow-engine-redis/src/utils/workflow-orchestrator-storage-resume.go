package utils

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/redis/go-redis/v9"

	"github.com/hitkey-io/medusago/packages/core/utils/src/bullmq"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// The resumption side of the storage: the checkpoint read like get(key),
// the step timeout jobs of the main queue and its worker. Go runs continue
// in the process that runs them (a finished async step does not add a
// retry job); a waiting step stops its run and the response resumes the
// transaction from the checkpoint (composer.DurableEngine).

var failedOrDoneStates = map[string]bool{
	composer.TransactionStateDone: true, composer.TransactionStateFailed: true, composer.TransactionStateReverted: true,
}

// LoadCheckpoint mirrors get(key): the latest workflow_execution row of the
// transaction (none, or a finished one: nil), with the flow and errors of
// the Redis checkpoint when present.
func (s *RedisDistributedTransactionStorage) LoadCheckpoint(ctx context.Context, workflowID, transactionID string) (*composer.Checkpoint, error) {
	cp, err := s.ReadCheckpoint(ctx, workflowID, transactionID)
	if err != nil || cp == nil {
		return nil, err
	}
	if state, _ := cp.Flow.Get("state"); failedOrDoneStates[fmt.Sprint(state)] {
		return nil, nil
	}
	return cp, nil
}

// ReadCheckpoint implements composer.CheckpointReader: the latest
// workflow_execution row of the transaction whatever its state, with the
// flow and errors of the Redis checkpoint when present.
func (s *RedisDistributedTransactionStorage) ReadCheckpoint(ctx context.Context, workflowID, transactionID string) (*composer.Checkpoint, error) {
	if s.opts.DB == nil {
		return nil, nil
	}
	rows, err := s.opts.DB.Query(ctx, `select "execution"::text, "context"::text from "workflow_execution"
where "workflow_id" = $1 and "transaction_id" = $2 and "deleted_at" is null order by "id" desc limit 1`, workflowID, transactionID)
	if err != nil {
		return nil, dal.MapDBError(err)
	}
	var execution, context_ *string
	found := false
	for rows.Next() {
		if err := rows.Scan(&execution, &context_); err != nil {
			rows.Close()
			return nil, err
		}
		found = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, dal.MapDBError(err)
	}
	if !found {
		return nil, nil
	}
	flow := decodeRecord(execution)
	ctxRec := decodeRecord(context_)
	cp := &composer.Checkpoint{Flow: flow}
	if data, ok := ctxRec.Get("data"); ok {
		cp.Context, _ = data.(*dal.Record)
	}
	if errs, ok := ctxRec.Get("errors"); ok {
		cp.Errors, _ = errs.([]any)
	}
	raw, err := s.opts.Client.Get(ctx, s.opts.Client.KeyPrefix+KeyName(workflowID, transactionID)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}
	if raw != "" {
		stored := decodeRecord(&raw)
		if f, ok := stored.Get("flow"); ok {
			if fr, ok := f.(*dal.Record); ok {
				cp.Flow = fr
			}
		}
		if e, ok := stored.Get("errors"); ok {
			cp.Errors, _ = e.([]any)
		}
	}
	return cp, nil
}

func decodeRecord(s *string) *dal.Record {
	if s == nil || *s == "" {
		return dal.NewRecord()
	}
	v, err := dal.DecodeOrderedJSON([]byte(*s))
	if err != nil {
		return dal.NewRecord()
	}
	r, _ := v.(*dal.Record)
	if r == nil {
		return dal.NewRecord()
	}
	return r
}

// resumeLockTTL bounds a resumption lock (the response is applied and
// persisted within it).
const resumeLockTTL = 30 * time.Second

// resumeLockWait is how long a resumption waits for the lock.
const resumeLockWait = 10 * time.Second

var unlockScript = redis.NewScript(`if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`)

// LockTransaction implements composer.DurableEngine: one resumption of a
// transaction at a time across the instances.
func (s *RedisDistributedTransactionStorage) LockTransaction(ctx context.Context, workflowID, transactionID string) (func(), error) {
	key := s.opts.Client.KeyPrefix + KeyName(workflowID, transactionID) + ":resume"
	token := ulid.Make().String()
	deadline := time.Now().Add(resumeLockWait)
	for {
		ok, err := s.opts.Client.SetNX(ctx, key, token, resumeLockTTL).Result()
		if err != nil {
			return nil, err
		}
		if ok {
			return func() { _ = unlockScript.Run(context.WithoutCancel(ctx), s.opts.Client, []string{key}, token).Err() }, nil
		}
		if time.Now().After(deadline) {
			return nil, errors.New("Lock not acquired")
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// stepTimeoutJobID mirrors getJobId(STEP_TIMEOUT, transaction, step).
func stepTimeoutJobID(job composer.StepTimeoutJob) string {
	return JobTypeStepTimeout + ":" + job.WorkflowID + ":" + job.TransactionID + ":" + job.StepID
}

// ScheduleStepTimeout mirrors scheduleStepTimeout.
func (s *RedisDistributedTransactionStorage) ScheduleStepTimeout(ctx context.Context, job composer.StepTimeoutJob) error {
	_, err := s.queue.Add(ctx, JobTypeStepTimeout,
		bullmq.NewObj("workflowId", job.WorkflowID, "transactionId", job.TransactionID, "transactionMetadata", job.Metadata, "stepId", job.StepID),
		bullmq.NewObj("delay", job.Seconds*1000, "jobId", stepTimeoutJobID(job), "removeOnComplete", true))
	return err
}

// ClearStepTimeout mirrors clearStepTimeout (removeJob: only a job not
// started yet).
func (s *RedisDistributedTransactionStorage) ClearStepTimeout(ctx context.Context, job composer.StepTimeoutJob) error {
	j, err := s.queue.GetJob(ctx, stepTimeoutJobID(job))
	if err != nil || j == nil {
		return err
	}
	if j.AttemptsStarted == 0 {
		return j.Remove(ctx)
	}
	return nil
}

// allowedJobs mirrors the job names the main worker resumes.
var allowedJobs = map[string]bool{JobTypeRetry: true, JobTypeStepTimeout: true, JobTypeTransactionTimeout: true}

// processMainJob mirrors the main worker: retry, step and transaction
// timeout jobs resume the transaction (executeTransaction); legacy
// schedule jobs are removed.
func (s *RedisDistributedTransactionStorage) processMainJob(ctx context.Context, job *bullmq.Job) (any, error) {
	data, _ := job.Data.(*bullmq.Obj)
	s.logger.Debug(fmt.Sprintf("executing job %s from queue %s", job.Name, s.opts.QueueName))
	if allowedJobs[job.Name] {
		workflowID, _ := mustGet(data, "workflowId").(string)
		transactionID, _ := mustGet(data, "transactionId").(string)
		if err := composer.ResumeTransaction(ctx, s.opts.Container, workflowID, transactionID); err != nil {
			return nil, err
		}
	}
	if job.Name == JobTypeSchedule {
		jobID, _ := mustGet(data, "jobId").(string)
		return nil, s.Remove(jobID)
	}
	return nil, nil
}
