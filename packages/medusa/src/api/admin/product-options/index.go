// Package productoptions mirrors api/admin/product-options.
package productoptions

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultAdminProductOptionsFields mirrors defaultAdminProductOptionsFields.
var DefaultAdminProductOptionsFields = []string{"id", "title", "is_exclusive", "values.*", "products.*", "created_at", "updated_at", "metadata"}

// RetrieveConfig / ListConfig mirror the query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultAdminProductOptionsFields, Entity: "product_option"}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultAdminProductOptionsFields, DefaultLimit: 20, IsList: true, Entity: "product_option"}
)

// AdminGetProductOptionsParamsFields mirrors AdminGetProductOptionsParamsFields.
var AdminGetProductOptionsParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("title", utils.StringOrStrings().Optional()),
	zod.F("is_exclusive", commonvalidators.BooleanString().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetProductOptionsParams mirrors AdminGetProductOptionsParams.
var AdminGetProductOptionsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(20), Offset: utils.Num(0)}).
	Merge(AdminGetProductOptionsParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetProductOptionsParamsFields))

// Middlewares mirrors adminProductOptionRoutesMiddlewares (GET routes).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/product-options", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetProductOptionsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/product-options/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RefetchListRoute("product_option", "product_options")}
