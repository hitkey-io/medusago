// Package stocklocations mirrors api/admin/stock-locations.
package stocklocations

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultAdminStockLocationFields mirrors defaultAdminStockLocationFields.
var DefaultAdminStockLocationFields = []string{"id", "name", "metadata", "created_at", "updated_at", "address.id", "address.address_1", "address.address_2", "address.city", "address.country_code", "address.phone", "address.province", "address.postal_code", "address.metadata"}

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultAdminStockLocationFields, Entity: "stock_location"}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultAdminStockLocationFields, IsList: true, Entity: "stock_location"}
)

// AdminGetStockLocationsParamsDirectFields mirrors AdminGetStockLocationsParamsDirectFields.
var AdminGetStockLocationsParamsDirectFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("name", utils.StringOrStrings().Optional()),
	zod.F("address_id", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetStockLocationsParams mirrors AdminGetStockLocationsParams.
var AdminGetStockLocationsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(20), Offset: utils.Num(0)}).
	Merge(AdminGetStockLocationsParamsDirectFields).
	Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetStockLocationsParamsDirectFields)).
	Merge(zod.Object(zod.F("sales_channel_id", utils.StringOrStrings().Optional())))

// Middlewares mirrors adminStockLocationRoutesMiddlewares (reads).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/stock-locations", Middlewares: []medusahttp.MiddlewareFunction{
		httputils.ValidateAndTransformQuery(AdminGetStockLocationsParams, ListConfig),
		httputils.MaybeApplyLinkFilter(httputils.LinkFilterOptions{EntryPoint: "sales_channel_location", ResourceID: "stock_location_id", FilterableField: "sales_channel_id"}),
	}},
	{Methods: []string{"GET"}, Matcher: "/admin/stock-locations/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("stock_locations", "stock_locations")}
