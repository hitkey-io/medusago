// Package returnreasons mirrors api/admin/return-reasons.
package returnreasons

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
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
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/return-reasons", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetReturnReasonsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/return-reasons/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("return_reason", "return_reasons")}
