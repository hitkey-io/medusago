// Package services mirrors @medusajs/workflow-engine-inmemory/src/services.
package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/modules/workflow-engine-inmemory/src/models"
	"github.com/hitkey-io/medusago/packages/modules/workflow-engine-inmemory/src/utils"
)

// WorkflowsModuleService mirrors WorkflowsModuleService: the generated
// WorkflowExecution service plus InMemoryDistributedTransactionStorage: the
// execution storage (saveToDb / deleteFromDb), the async steps waiting in
// this process, the scheduler and the expired executions cleaner.
type WorkflowsModuleService struct {
	*utilsmodulessdk.MedusaService
	*composer.LocalStepCoordinator
	db        dal.Querier
	scheduler *utils.Scheduler
	logger    *slog.Logger
	stop      chan struct{}
}

// NewWorkflowsModuleService builds the service.
func NewWorkflowsModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*WorkflowsModuleService, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	var c composer.Container
	if opts.Container != nil {
		c = opts.Container
	}
	return &WorkflowsModuleService{
		MedusaService:        utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig),
		LocalStepCoordinator: composer.NewLocalStepCoordinator(),
		db:                   opts.DB,
		scheduler:            utils.NewScheduler(c, logger),
		logger:               logger,
	}, nil
}

var (
	_ composer.ExecutionStore    = (*WorkflowsModuleService)(nil)
	_ composer.StepCoordinator   = (*WorkflowsModuleService)(nil)
	_ composer.WorkflowScheduler = (*WorkflowsModuleService)(nil)
)

// OnApplicationStart mirrors onApplicationStart: the expired executions are
// cleared every 30 minutes; the scheduled workflows are scheduled.
func (s *WorkflowsModuleService) OnApplicationStart(context.Context) error {
	if s.stop != nil {
		return nil
	}
	s.stop = make(chan struct{})
	go func(stop chan struct{}) {
		t := time.NewTicker(utils.ThirtyMinutes)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				_ = utils.ClearExpiredExecutions(context.Background(), s.db)
			}
		}
	}(s.stop)
	return composer.StartScheduler(s)
}

// OnApplicationShutdown mirrors onApplicationShutdown: the timers stop.
func (s *WorkflowsModuleService) OnApplicationShutdown(context.Context) error {
	composer.StopScheduler(s)
	s.scheduler.RemoveAll()
	if s.stop != nil {
		close(s.stop)
		s.stop = nil
	}
	return nil
}

// Schedule mirrors schedule(jobDefinition, schedulerOptions).
func (s *WorkflowsModuleService) Schedule(jobID string, opts composer.SchedulerOptions) error {
	return s.scheduler.Schedule(jobID, opts)
}

// Remove mirrors remove(jobId).
func (s *WorkflowsModuleService) Remove(jobID string) error { return s.scheduler.Remove(jobID) }

// ClearExpiredExecutions mirrors clearExpiredExecutions.
func (s *WorkflowsModuleService) ClearExpiredExecutions(ctx context.Context) error {
	return utils.ClearExpiredExecutions(ctx, s.db)
}

// SaveExecution mirrors workflowExecutionService.upsert([{ workflow_id,
// transaction_id, run_id, execution, context, state, retention_time }]).
func (s *WorkflowsModuleService) SaveExecution(ctx context.Context, row composer.ExecutionRow) error {
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
	_, err = s.db.Exec(ctx, `insert into "workflow_execution" ("id", "workflow_id", "transaction_id", "run_id", "execution", "context", "state", "retention_time", "created_at", "updated_at")
values ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7, $8, $9, $9)
on conflict ("workflow_id", "transaction_id", "run_id") do update set "execution" = excluded."execution", "context" = excluded."context",
  "state" = excluded."state", "retention_time" = excluded."retention_time", "updated_at" = excluded."updated_at"`,
		common.GenerateEntityID("", "wf_exec"), row.WorkflowID, row.TransactionID, row.RunID, string(execution), string(context_), row.State, retention, now)
	return dal.MapDBError(err)
}

// DeleteExecution mirrors workflowExecutionService.delete([{ run_id }]).
func (s *WorkflowsModuleService) DeleteExecution(ctx context.Context, workflowID, transactionID, runID string) error {
	_, err := s.db.Exec(ctx, `delete from "workflow_execution" where "run_id" = $1`, runID)
	return dal.MapDBError(err)
}
