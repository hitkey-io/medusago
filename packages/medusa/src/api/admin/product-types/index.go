// Package producttypes mirrors api/admin/product-types.
package producttypes

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultAdminProductTypeFields mirrors defaultAdminProductTypeFields.
var DefaultAdminProductTypeFields = []string{"id", "value", "external_id", "created_at", "updated_at", "metadata"}

// RetrieveConfig / ListConfig mirror the query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultAdminProductTypeFields, Entity: "product_type"}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultAdminProductTypeFields, DefaultLimit: 20, IsList: true, Entity: "product_type"}
)

// AdminGetProductTypesParamsFields mirrors AdminGetProductTypesParamsFields.
var AdminGetProductTypesParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("value", utils.StringOrStrings().Optional()),
	zod.F("external_id", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetProductTypesParams mirrors AdminGetProductTypesParams.
var AdminGetProductTypesParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(10), Offset: utils.Num(0)}).
	Merge(AdminGetProductTypesParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetProductTypesParamsFields))

// Middlewares mirrors adminProductTypeRoutesMiddlewares (GET routes).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/product-types", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetProductTypesParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/product-types/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("product_type", "product_types")}
