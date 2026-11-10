// Package services mirrors @medusajs/workflow-engine-inmemory/src/services.
package services

import (
	"context"
	"encoding/json"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/modules/workflow-engine-inmemory/src/models"
)

// WorkflowsModuleService mirrors WorkflowsModuleService: the generated
// WorkflowExecution service plus the execution storage of
// InMemoryDistributedTransactionStorage (saveToDb / deleteFromDb).
type WorkflowsModuleService struct {
	*utilsmodulessdk.MedusaService
	db dal.Querier
}

// NewWorkflowsModuleService builds the service.
func NewWorkflowsModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*WorkflowsModuleService, error) {
	return &WorkflowsModuleService{MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig), db: opts.DB}, nil
}

var _ composer.ExecutionStore = (*WorkflowsModuleService)(nil)

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
