// Package collections mirrors api/store/collections.
package collections

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/store/products"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultStoreCollectionFields mirrors defaultStoreCollectionFields.
var DefaultStoreCollectionFields = []string{"id", "title", "handle", "external_id", "created_at", "updated_at"}

var allowed = storeutils.BuildAllowedFields(DefaultStoreCollectionFields, append([]string{"products", "metadata", "deleted_at"},
	storeutils.PrefixAllowedFields("products", products.DefaultStoreProductFields, products.AllowedStoreProductExtraFields)...))

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultStoreCollectionFields, Allowed: allowed}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultStoreCollectionFields, Allowed: allowed, DefaultLimit: 10, IsList: true}
)

// StoreGetCollectionsParamsFields mirrors StoreGetCollectionsParamsFields.
var StoreGetCollectionsParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("title", utils.StringOrStrings().Optional()),
	zod.F("handle", utils.StringOrStrings().Optional()),
	zod.F("external_id", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// StoreGetCollectionsParams mirrors StoreGetCollectionsParams.
var StoreGetCollectionsParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(10), Order: "-created_at"}).
	Merge(StoreGetCollectionsParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(StoreGetCollectionsParamsFields))

// Middlewares mirrors storeCollectionRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/store/collections", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(StoreGetCollectionsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/store/collections/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: storeutils.GraphListRoute("product_collection", "collections", true, false)}
