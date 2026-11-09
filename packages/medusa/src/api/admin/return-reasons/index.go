// Package returnreasons mirrors api/admin/return-reasons.
package returnreasons

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/return-reason/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// Default field lists.
var (
	DefaultAdminReturnReasonFields         = []string{"id", "value", "label", "parent_return_reason_id", "description", "created_at", "updated_at", "deleted_at"}
	DefaultAdminRetrieveReturnReasonFields = append(append([]string{}, DefaultAdminReturnReasonFields...), "parent_return_reason.*", "return_reason_children.*")
)

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultAdminRetrieveReturnReasonFields, Entity: "return_reason"}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultAdminReturnReasonFields, DefaultLimit: 20, IsList: true, Entity: "return_reason"}
)

// AdminGetReturnReasonsParamsFields mirrors AdminGetReturnReasonsParamsFields.
var AdminGetReturnReasonsParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("value", utils.StringOrStrings().Optional()),
	zod.F("label", utils.StringOrStrings().Optional()),
	zod.F("description", utils.StringOrStrings().Optional()),
	zod.F("parent_return_reason_id", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetReturnReasonsParams mirrors AdminGetReturnReasonsParams.
var AdminGetReturnReasonsParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(20)}).
	Merge(AdminGetReturnReasonsParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetReturnReasonsParamsFields))

// Middlewares mirrors adminReturnReasonRoutesMiddlewares (reads).
// AdminCreateReturnReason mirrors AdminCreateReturnReason.
var AdminCreateReturnReason = zod.Object(
	zod.F("value", zod.String()),
	zod.F("label", zod.String()),
	zod.F("description", zod.String().Nullish()),
	zod.F("parent_return_reason_id", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

// AdminUpdateReturnReason mirrors AdminUpdateReturnReason.
var AdminUpdateReturnReason = zod.Object(
	zod.F("value", zod.String()),
	zod.F("label", zod.String()),
	zod.F("description", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/return-reasons", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetReturnReasonsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/return-reasons/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/return-reasons", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateReturnReason)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/return-reasons/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateReturnReason)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("return_reason", "return_reasons"), POST: POST}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	result, err := workflows.CreateReturnReasonsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{body})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	return RespondByID(req, res, id)
}

// RespondByID mirrors remoteQuery({ entryPoint: "return_reason", variables:
// { id } })[0] written as { return_reason }.
func RespondByID(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id any) error {
	vars := dal.NewRecord()
	vars.Set("id", id)
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "return_reason", vars, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if len(r.Data) > 0 {
		out.Set("return_reason", r.Data[0])
	}
	return res.JSON(out)
}
