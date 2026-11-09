// Package collections mirrors api/admin/collections.
package collections

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultAdminCollectionFields mirrors defaultAdminCollectionFields.
var DefaultAdminCollectionFields = []string{"id", "title", "handle", "external_id", "created_at", "updated_at", "metadata"}

// RetrieveConfig / ListConfig mirror the query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultAdminCollectionFields, Entity: "product_collection"}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultAdminCollectionFields, DefaultLimit: 10, IsList: true, Entity: "product_collection"}
)

// AdminGetCollectionsParamsFields mirrors AdminGetCollectionsParamsFields.
var AdminGetCollectionsParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("title", utils.StringOrStrings().Optional()),
	zod.F("handle", utils.StringOrStrings().Optional()),
	zod.F("external_id", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetCollectionsParams mirrors AdminGetCollectionsParams.
var AdminGetCollectionsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(10), Offset: utils.Num(0)}).
	Merge(AdminGetCollectionsParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetCollectionsParamsFields))

// Middlewares mirrors adminCollectionRoutesMiddlewares (authorize entries are Enterprise and skipped).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/collections", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetCollectionsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/collections/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/collections", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateCollection)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/collections/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateCollection)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("product_collection", "collections"), POST: POST}
