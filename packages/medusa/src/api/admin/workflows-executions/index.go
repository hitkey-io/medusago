// Package workflowsexecutions mirrors api/admin/workflows-executions.
package workflowsexecutions

import (
	"strings"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// Field lists.
var (
	DefaultAdminWorkflowExecutionDetailFields = []string{"id", "workflow_id", "transaction_id", "context", "execution", "state", "created_at", "updated_at", "deleted_at"}
	DefaultAdminWorkflowExecutionsFields      = []string{"id", "workflow_id", "transaction_id", "state", "execution", "created_at", "updated_at", "deleted_at"}
)

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultAdminWorkflowExecutionDetailFields, Entity: "workflow_execution"}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultAdminWorkflowExecutionsFields, IsList: true, Entity: "workflow_execution"}
)

// AdminGetWorkflowExecutionsParams mirrors AdminGetWorkflowExecutionsParams.
var AdminGetWorkflowExecutionsParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(100)}).Merge(zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("transaction_id", utils.StringOrStrings().Optional()),
	zod.F("workflow_id", utils.StringOrStrings().Optional()),
	zod.F("state", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
))

// Middlewares mirrors adminWorkflowsExecutionsMiddlewares (reads).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/workflows-executions", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetWorkflowExecutionsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/workflows-executions/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/workflows-executions/:workflow_id/:transaction_id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/workflows-executions/:workflow_id/steps/success", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateWorkflowsAsyncResponse))}},
	{Methods: []string{"POST"}, Matcher: "/admin/workflows-executions/:workflow_id/steps/failure", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateWorkflowsAsyncResponse))}},
}

// AdminCreateWorkflowsAsyncResponse mirrors AdminCreateWorkflowsAsyncResponse.
var AdminCreateWorkflowsAsyncResponse = zod.Object(
	zod.F("transaction_id", zod.String()),
	zod.F("step_id", zod.String()),
	zod.F("response", zod.Any().Optional()),
	zod.F("compensate_input", zod.Any().Optional()),
	zod.F("action", zod.Preprocess(func(v any) any { return strings.ToLower(jsString(v)) }, zod.Enum("invoke", "compensate").Optional()).Optional()),
)

// jsString mirrors val + "".
func jsString(v any) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return totals.JSNumberToString(t)
	}
	return "[object Object]"
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("workflow_execution", "workflow_executions")}
