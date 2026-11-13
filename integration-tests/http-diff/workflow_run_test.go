package httpdiff

import (
	"fmt"

	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	orderworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// The test-only /workflow-run route of the reference application
// (workflow-run-route.js): core workflows run as a step followed by a
// failing step, so that the core workflow is compensated after it finished.

type httpDiffFailInput struct {
	After string `json:"after"`
}

var httpDiffFailStep = composer.CreateStep("http-diff-fail",
	func(ctx composer.StepExecutionContext, in httpDiffFailInput) (composer.StepResponse[any], error) {
		return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Forced failure after "+in.After)
	}, nil)

type httpDiffRunInput struct {
	ID          any           `json:"id"`
	CartID      string        `json:"cart_id"`
	CreditLines []*dal.Record `json:"credit_lines"`
}

var httpDiffRunWorkflows = map[string]*composer.Workflow[httpDiffRunInput, any]{
	"http-diff-complete-cart-then-fail": composer.CreateWorkflow("http-diff-complete-cart-then-fail",
		func(e *composer.Execution, in httpDiffRunInput) (any, error) {
			id, _ := in.ID.(string)
			if _, err := cartworkflows.CompleteCartWorkflow.RunAsStep(e, cartworkflows.CompleteCartWorkflowInput{ID: id}); err != nil {
				return nil, err
			}
			_, err := httpDiffFailStep.Run(e, httpDiffFailInput{After: "complete-cart"})
			return nil, err
		}),
	"http-diff-credit-lines-then-fail": composer.CreateWorkflow("http-diff-credit-lines-then-fail",
		func(e *composer.Execution, in httpDiffRunInput) (any, error) {
			if _, err := orderworkflows.CreateOrderCreditLinesWorkflow.RunAsStep(e, struct {
				ID          any
				CreditLines []*dal.Record
			}{ID: in.ID, CreditLines: in.CreditLines}); err != nil {
				return nil, err
			}
			_, err := httpDiffFailStep.Run(e, httpDiffFailInput{After: "create-order-credit-lines"})
			return nil, err
		}),
	"http-diff-refresh-cart-then-fail": composer.CreateWorkflow("http-diff-refresh-cart-then-fail",
		func(e *composer.Execution, in httpDiffRunInput) (any, error) {
			if _, err := cartworkflows.RefreshCartItemsWorkflow.RunAsStep(e, cartworkflows.RefreshCartItemsInput{CartID: in.CartID}); err != nil {
				return nil, err
			}
			_, err := httpDiffFailStep.Run(e, httpDiffFailInput{After: "refresh-cart-items"})
			return nil, err
		}),
}

func workflowRunRoutes() []medusahttp.RouteFile {
	return []medusahttp.RouteFile{{Path: "/workflow-run", Route: medusahttp.Route{POST: workflowRunRoute}}}
}

func workflowRunRoute(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.Body.(*dal.Record)
	if body == nil {
		body = dal.NewRecord()
	}
	v, _ := body.Get("workflow_id")
	workflowID, _ := v.(string)
	w := httpDiffRunWorkflows[workflowID]
	if w == nil {
		out := dal.NewRecord()
		out.Set("error", fmt.Sprintf("unknown workflow %s", workflowID))
		return res.Status(400).JSON(out)
	}
	rawInput, _ := body.Get("input")
	var input httpDiffRunInput
	if err := composer.DecodeInput(rawInput, &input); err != nil {
		return err
	}
	_, info, err := w.RunWithInfo(req.Ctx(), req.Scope, input)
	out := dal.NewRecord()
	out.Set("state", info.State)
	if err != nil {
		e := dal.NewRecord()
		if me, ok := common.AsMedusaError(err); ok {
			e.Set("message", me.Message)
			e.Set("type", me.Type)
		} else {
			e.Set("message", err.Error())
			e.Set("type", nil)
		}
		out.Set("error", e)
	} else {
		out.Set("error", nil)
	}
	return res.JSON(out)
}
