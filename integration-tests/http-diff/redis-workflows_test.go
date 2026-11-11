package httpdiff

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// The http-diff-wait workflows and their route (HTTP_DIFF_WAIT_WORKFLOW=true),
// mirroring testapp/src/workflows/http-diff-wait.js and
// testapp/src/api/admin/http-diff-wait/[transaction_id]/route.js of the
// reference application: stored workflows waiting for an async step (step
// timeout 60 s, or 2 s for http-diff-wait-timeout) between two steps
// creating product tags.

type httpDiffWaitInput struct {
	Value string `json:"value"`
}

func createOneTag(ctx context.Context, c composer.Container, data *dal.Record) (*dal.Record, error) {
	svc, ok := c.Resolve("product").(tagCreator)
	if !ok {
		return nil, errors.New("product module not found")
	}
	created, err := svc.CreateProductTags(ctx, []*dal.Record{data})
	if err != nil {
		return nil, err
	}
	if len(created) == 0 {
		return nil, errors.New("no tag created")
	}
	return created[0], nil
}

func tagRef(tag *dal.Record) *dal.Record {
	id, _ := tag.Get("id")
	value, _ := tag.Get("value")
	r := dal.NewRecord()
	r.Set("id", id)
	r.Set("value", value)
	return r
}

var httpDiffWaitBefore = composer.CreateStep("http-diff-wait-before",
	func(ctx composer.StepExecutionContext, in httpDiffWaitInput) (composer.StepResponse[*dal.Record], error) {
		data := dal.NewRecord()
		data.Set("value", in.Value+"-before")
		tag, err := createOneTag(ctx.Context, ctx.Container, data)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		id, _ := tag.Get("id")
		return composer.NewStepResponse(tagRef(tag), id), nil
	},
	func(ctx composer.StepExecutionContext, id any) error {
		if id == nil || id == "" {
			return nil
		}
		_, err := steps.ProductService(ctx.Container).Internal("ProductTag").SoftDelete(ctx.Context, []any{id})
		return err
	})

var httpDiffWaitStep = composer.CreateStep("http-diff-wait-step",
	func(composer.StepExecutionContext, any) (composer.StepResponse[any], error) {
		return composer.NewStepResponse[any](nil), nil
	}, nil).Config("", composer.StepOptions{Async: true, Timeout: 60})

var httpDiffShortWaitStep = httpDiffWaitStep.Config("", composer.StepOptions{Async: true, Timeout: 2})

type httpDiffWaitAfterInput struct {
	Before   *dal.Record
	Response any
}

var httpDiffWaitAfter = composer.CreateStep("http-diff-wait-after",
	func(ctx composer.StepExecutionContext, in httpDiffWaitAfterInput) (composer.StepResponse[*dal.Record], error) {
		value, _ := in.Before.Get("value")
		md := dal.NewRecord()
		md.Set("response", in.Response)
		data := dal.NewRecord()
		data.Set("value", fmt.Sprint(value)+"-after")
		data.Set("metadata", md)
		tag, err := createOneTag(ctx.Context, ctx.Container, data)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(tagRef(tag)), nil
	}, nil)

func buildHTTPDiffWait(name string, wait *composer.Step[any, any]) *composer.Workflow[httpDiffWaitInput, *dal.Record] {
	return composer.CreateWorkflowWithOptions(name, composer.WorkflowOptions{Store: true, RetentionTime: 3600},
		func(e *composer.Execution, in httpDiffWaitInput) (*dal.Record, error) {
			before, err := httpDiffWaitBefore.Run(e, in)
			if err != nil {
				return nil, err
			}
			response, err := wait.Run(e, nil)
			if err != nil {
				return nil, err
			}
			after, err := httpDiffWaitAfter.Run(e, httpDiffWaitAfterInput{Before: before, Response: response})
			if err != nil {
				return nil, err
			}
			out := dal.NewRecord()
			out.Set("before", before)
			out.Set("after", after)
			return out, nil
		})
}

var (
	httpDiffWaitWorkflow        = buildHTTPDiffWait("http-diff-wait", httpDiffWaitStep)
	httpDiffWaitTimeoutWorkflow = buildHTTPDiffWait("http-diff-wait-timeout", httpDiffShortWaitStep)
)

var _, _ = httpDiffWaitWorkflow, httpDiffWaitTimeoutWorkflow

// httpDiffWaitRoute mirrors api/admin/http-diff-wait/[transaction_id]/route.js.
func httpDiffWaitRoute(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.Body.(*dal.Record)
	if body == nil {
		body = dal.NewRecord()
	}
	workflowID := "http-diff-wait"
	if v, _ := body.Get("workflow_id"); v != nil {
		workflowID = fmt.Sprint(v)
	}
	key := composer.IdempotencyKey{Action: "invoke", TransactionID: req.Params["transaction_id"], StepID: "http-diff-wait-step", WorkflowID: workflowID}
	action, _ := body.Get("action")
	response, _ := body.Get("response")
	var err error
	if action == "failure" {
		err = composer.SetStepFailure(req.Ctx(), req.Scope, key, errors.New(fmt.Sprint(response)))
	} else {
		err = composer.SetStepSuccess(req.Ctx(), req.Scope, key, response)
	}
	if err != nil {
		out := dal.NewRecord()
		var me *common.MedusaError
		if errors.As(err, &me) {
			out.Set("type", me.Type)
		} else {
			out.Set("type", nil)
		}
		out.Set("message", err.Error())
		return res.Status(400).JSON(out)
	}
	ok := dal.NewRecord()
	ok.Set("ok", true)
	return res.JSON(ok)
}

// scenarioRoutes are the routes of the reference application's src/api
// enabled for the scenario.
func scenarioRoutes() []medusahttp.RouteFile {
	if os.Getenv("HTTP_DIFF_WAIT_WORKFLOW") != "true" {
		return nil
	}
	return []medusahttp.RouteFile{
		{Path: "/admin/http-diff-wait/:transaction_id", Route: medusahttp.Route{POST: httpDiffWaitRoute}},
	}
}
