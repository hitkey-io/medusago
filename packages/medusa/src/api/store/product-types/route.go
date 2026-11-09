// Package producttypes mirrors api/store/product-types.
package producttypes

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/store/products"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// Defaults mirrors defaults.
var Defaults = []string{"id", "value", "external_id", "created_at", "updated_at", "metadata", "*products"}

var allowed = storeutils.BuildAllowedFields(Defaults, storeutils.PrefixAllowedFields("products", products.DefaultStoreProductFields, products.AllowedStoreProductExtraFields))

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: Defaults, Allowed: allowed}
	ListConfig     = httputils.QueryConfig{Defaults: Defaults, Allowed: allowed, DefaultLimit: 50, IsList: true}
)

// ParamsFields mirrors StoreProductTypesParamsFields.
var ParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("value", utils.StringOrStrings().Optional()),
	zod.F("external_id", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// ListParams mirrors StoreProductTypesParams.
var ListParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(50)}).
	Merge(ParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(ParamsFields))

// Middlewares mirrors storeProductTypeRoutesMiddlewares (the retrieve route
// validates with the list params upstream).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/store/product-types", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(ListParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/store/product-types/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(ListParams, RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: storeutils.GraphListRoute("product_type", "product_types", true, true)}
