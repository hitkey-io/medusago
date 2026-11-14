package store

import (
	"context"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// workflowEngine is the Workflow Engine Module's service.
type workflowEngine interface {
	Call(ctx context.Context, method string, args []any) (any, error)
}

// region: retrieve
// GET retrieves the stored execution of a transaction. Its route is
// /workflows/:id.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	engine := req.Scope.Resolve(utilsmodulessdk.WORKFLOW_ENGINE).(workflowEngine)

	filters := dal.NewRecord()
	filters.Set("transaction_id", req.Params["id"])

	executions, err := engine.Call(req.Ctx(), "listWorkflowExecutions", []any{filters})
	if err != nil {
		return err
	}

	var workflowExecution *dal.Record
	if list := executions.([]*dal.Record); len(list) > 0 {
		workflowExecution = list[0]
	}

	out := dal.NewRecord()
	out.Set("workflowExecution", workflowExecution)
	return res.JSON(out)
}

// endregion: retrieve

// region: state
// Failed reports whether a stored workflow execution failed.
func Failed(workflowExecution *dal.Record) bool {
	state, _ := workflowExecution.Get("state")
	return state == "failed"
}

// endregion: state

// region: transaction-id
// RunWithTransactionID runs the workflow with a transaction ID of your own.
// Without one, a unique ID is generated for every execution.
func RunWithTransactionID(ctx context.Context, container composer.Container) error {
	_, err := HelloWorkflow.Run(ctx, container, nil, composer.RunOptions{
		TransactionID: "hello-123",
	})
	return err
}

// endregion: transaction-id

// region: info
// RunAndGetTransaction runs the workflow and returns the details of its
// transaction.
func RunAndGetTransaction(ctx context.Context, container composer.Container) (composer.RunInfo, error) {
	_, info, err := HelloWorkflow.RunWithInfo(ctx, container, nil)
	// info.TransactionID, info.RunID and info.State
	return info, err
}

// endregion: info
