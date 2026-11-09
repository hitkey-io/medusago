// Package refundreasons mirrors api/admin/refund-reasons.
package refundreasons

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/payment-collection/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// DefaultAdminRefundReasonFields mirrors defaultAdminRefundReasonFields.
var DefaultAdminRefundReasonFields = []string{"id", "label", "code", "description", "created_at", "updated_at", "deleted_at"}

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultAdminRefundReasonFields, Entity: "refund_reason"}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultAdminRefundReasonFields, DefaultLimit: 20, IsList: true, Entity: "refund_reason"}
)

// AdminGetRefundReasonsParams mirrors AdminGetRefundReasonsParams.
var AdminGetRefundReasonsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(15), Offset: utils.Num(0)}).Merge(zod.Object(
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("q", zod.String().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
))

// Middlewares mirrors adminRefundReasonsRoutesMiddlewares (reads).
// AdminCreatePaymentRefundReason mirrors AdminCreatePaymentRefundReason.
var AdminCreatePaymentRefundReason = zod.Object(
	zod.F("label", zod.String()),
	zod.F("code", zod.String()),
	zod.F("description", zod.String().Nullish()),
).Strict()

// AdminUpdatePaymentRefundReason mirrors AdminUpdatePaymentRefundReason.
var AdminUpdatePaymentRefundReason = zod.Object(
	zod.F("label", zod.String().Optional()),
	zod.F("code", zod.String().Optional()),
	zod.F("description", zod.String().Nullish()),
).Strict()

var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/refund-reasons", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetRefundReasonsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/refund-reasons/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetRefundReasonsParams, RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/refund-reasons", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminCreatePaymentRefundReason)), httputils.ValidateAndTransformQuery(AdminGetRefundReasonsParams, RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/refund-reasons/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdatePaymentRefundReason)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
	{Methods: []string{"DELETE"}, Matcher: "/admin/refund-reasons/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetRefundReasonsParams, RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RefetchListRoute("refund_reasons", "refund_reasons"), POST: POST}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	result, err := workflows.CreateRefundReasonsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{body})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	return Respond(req, res, id)
}

// Respond refetches the refund reason and writes { refund_reason }.
func Respond(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id any) error {
	r, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "refund_reason", id, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if r != nil {
		out.Set("refund_reason", r)
	}
	return res.Status(200).JSON(out)
}
