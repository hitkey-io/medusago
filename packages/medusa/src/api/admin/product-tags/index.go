// Package producttags mirrors api/admin/product-tags.
package producttags

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultAdminProductTagFields mirrors defaultAdminProductTagFields.
var DefaultAdminProductTagFields = []string{"id", "value", "external_id", "created_at", "updated_at", "metadata"}

// RetrieveConfig / ListConfig mirror the query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultAdminProductTagFields, Entity: "product_tag"}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultAdminProductTagFields, DefaultLimit: 20, IsList: true, Entity: "product_tag"}
)

// AdminGetProductTagsParamsFields mirrors AdminGetProductTagsParamsFields.
var AdminGetProductTagsParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("value", utils.StringOrStrings().Optional()),
	zod.F("external_id", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetProductTagsParams mirrors AdminGetProductTagsParams.
var AdminGetProductTagsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(20), Offset: utils.Num(0)}).
	Merge(AdminGetProductTagsParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetProductTagsParamsFields))

// Middlewares mirrors adminProductTagRoutesMiddlewares (authorize entries are Enterprise and skipped).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/product-tags", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetProductTagsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/product-tags/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/product-tags", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateProductTag)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/product-tags/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateProductTag)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RefetchListRoute("product_tag", "product_tags"), POST: POST}
