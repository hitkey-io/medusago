// Package regions mirrors api/store/regions.
package regions

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultStoreRegionFields mirrors defaultStoreRegionFields.
var DefaultStoreRegionFields = []string{"id", "name", "currency_code", "created_at", "updated_at", "deleted_at", "metadata", "*countries"}

var allowed = storeutils.BuildAllowedFields(DefaultStoreRegionFields,
	[]string{"countries.id", "countries.iso_2", "countries.iso_3", "countries.num_code", "countries.name", "countries.display_name"},
	append([]string{"payment_providers"}, storeutils.PrefixAllowedFields("payment_providers", []string{"id", "is_enabled"})...))

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultStoreRegionFields, Allowed: allowed}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultStoreRegionFields, Allowed: allowed, DefaultLimit: 20, IsList: true}
)

// StoreGetRegionsParamsFields mirrors StoreGetRegionsParamsFields.
var StoreGetRegionsParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("currency_code", utils.StringOrStrings().Optional()),
	zod.F("name", utils.StringOrStrings().Optional()),
)

// StoreGetRegionsParams mirrors StoreGetRegionsParams.
var StoreGetRegionsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(50), Offset: utils.Num(0)}).
	Merge(StoreGetRegionsParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(StoreGetRegionsParamsFields))

// Middlewares mirrors storeRegionRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/store/regions", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(StoreGetRegionsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/store/regions/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("region", "regions")}
