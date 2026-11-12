package httpdiff

import (
	"context"
	"fmt"
	"sort"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// workflowEngineOpsRoutes adds the test-only /workflow-engine-ops route of
// the reference application (workflow-engine-route.js): it runs
// getRunningTransaction / cancel / retryStep of the workflow engine module
// and answers a summary of the transaction flow.
func workflowEngineOpsRoutes() []medusahttp.RouteFile {
	return []medusahttp.RouteFile{{Path: "/workflow-engine-ops", Route: medusahttp.Route{POST: workflowEngineOpsRoute}}}
}

func workflowEngineOpsRoute(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.Body.(*dal.Record)
	if body == nil {
		body = dal.NewRecord()
	}
	str := func(k string) string {
		v, _ := body.Get(k)
		s, _ := v.(string)
		return s
	}
	engine, ok := req.Scope.Resolve(utilsmodulessdk.WORKFLOW_ENGINE).(interface {
		GetRunningTransaction(ctx context.Context, workflowID, transactionID string) (*composer.Checkpoint, error)
		Cancel(ctx context.Context, workflowID string, opts composer.CancelOptions) (*composer.CancelResult, error)
		RetryStep(ctx context.Context, key composer.IdempotencyKey) (*composer.Checkpoint, error)
	})
	if !ok {
		return fmt.Errorf("the workflow engine module does not expose cancel / retryStep")
	}
	fail := func(err error) error {
		out := dal.NewRecord()
		out.Set("error", err.Error())
		if me, ok := common.AsMedusaError(err); ok {
			out.Set("type", me.Type)
		} else {
			out.Set("type", nil)
		}
		return res.Status(400).JSON(out)
	}
	out := dal.NewRecord()
	switch str("op") {
	case "get":
		cp, err := engine.GetRunningTransaction(req.Ctx(), str("workflow_id"), str("transaction_id"))
		if err != nil {
			return fail(err)
		}
		out.Set("transaction", transactionSummary(cp))
	case "cancel":
		opts := composer.CancelOptions{TransactionID: str("transaction_id")}
		if v, ok := body.Get("throw_on_error"); ok {
			if b, ok := v.(bool); ok {
				opts.ThrowOnError = &b
			}
		}
		r, err := engine.Cancel(req.Ctx(), str("workflow_id"), opts)
		if err != nil {
			return fail(err)
		}
		out.Set("acknowledgement", r.Acknowledgement)
		out.Set("transaction", transactionSummary(r.Transaction))
	case "retry":
		cp, err := engine.RetryStep(req.Ctx(), composer.IdempotencyKey{Action: "invoke", TransactionID: str("transaction_id"), StepID: str("step_id"), WorkflowID: str("workflow_id")})
		if err != nil {
			return fail(err)
		}
		out.Set("transaction", transactionSummary(cp))
	default:
		e := dal.NewRecord()
		e.Set("error", "unknown op "+str("op"))
		return res.Status(400).JSON(e)
	}
	return res.JSON(out)
}

// transactionSummary mirrors summary() of workflow-engine-route.js.
func transactionSummary(cp *composer.Checkpoint) any {
	if cp == nil {
		return nil
	}
	get := func(r *dal.Record, k string) any {
		if r == nil {
			return nil
		}
		v, _ := r.Get(k)
		return v
	}
	rec := func(v any) *dal.Record { r, _ := v.(*dal.Record); return r }
	flow := cp.Flow
	steps := dal.NewRecord()
	if all := rec(get(flow, "steps")); all != nil {
		ids := append([]string{}, all.Keys()...)
		sort.Strings(ids)
		for _, id := range ids {
			if id == "_root" {
				continue
			}
			st := rec(get(all, id))
			s := dal.NewRecord()
			s.Set("invoke", get(rec(get(st, "invoke")), "state"))
			s.Set("invoke_status", get(rec(get(st, "invoke")), "status"))
			s.Set("compensate", get(rec(get(st, "compensate")), "state"))
			steps.Set(id, s)
		}
	}
	out := dal.NewRecord()
	out.Set("state", get(flow, "state"))
	cancelled := get(flow, "cancelledAt")
	out.Set("cancelled", cancelled != nil && cancelled != float64(0))
	out.Set("has_failed_steps", get(flow, "hasFailedSteps") == true)
	out.Set("steps", steps)
	return out
}
