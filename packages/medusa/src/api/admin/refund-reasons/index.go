// Package refundreasons mirrors api/admin/refund-reasons.
package refundreasons

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
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
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/refund-reasons", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetRefundReasonsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/refund-reasons/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetRefundReasonsParams, RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RefetchListRoute("refund_reasons", "refund_reasons")}
