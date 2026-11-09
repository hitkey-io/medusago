// Package inventoryitems mirrors api/admin/inventory-items.
package inventoryitems

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// Default field lists.
var (
	DefaultAdminLocationLevelFields = []string{"id", "inventory_item_id", "location_id", "stocked_quantity", "reserved_quantity", "incoming_quantity", "available_quantity", "metadata", "created_at", "updated_at"}
	DefaultAdminInventoryItemFields = []string{"id", "sku", "title", "description", "thumbnail", "origin_country", "hs_code", "requires_shipping", "mid_code", "material", "unit_of_measure", "weight", "length", "height", "width", "metadata", "reserved_quantity", "stocked_quantity", "created_at", "updated_at", "*location_levels"}
)

// Query configs.
var (
	RetrieveConfig              = httputils.QueryConfig{Defaults: DefaultAdminInventoryItemFields, Entity: "inventory_item"}
	ListConfig                  = httputils.QueryConfig{Defaults: DefaultAdminInventoryItemFields, IsList: true, Entity: "inventory_item"}
	RetrieveLocationLevelConfig = httputils.QueryConfig{Defaults: DefaultAdminLocationLevelFields, Entity: "inventory_level"}
	ListLocationLevelsConfig    = httputils.QueryConfig{Defaults: DefaultAdminLocationLevelFields, IsList: true, Entity: "inventory_level"}
)

func numberOperatorMap() *zod.Schema { return utils.CreateOperatorMap(zod.Number(), utils.ParseFloat) }

// AdminGetInventoryItemsParamsFields mirrors AdminGetInventoryItemsParamsFields.
var AdminGetInventoryItemsParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("sku", utils.StringOrStrings().Optional()),
	zod.F("origin_country", utils.StringOrStrings().Optional()),
	zod.F("mid_code", utils.StringOrStrings().Optional()),
	zod.F("hs_code", utils.StringOrStrings().Optional()),
	zod.F("material", utils.StringOrStrings().Optional()),
	zod.F("requires_shipping", commonvalidators.BooleanString().Optional()),
	zod.F("weight", numberOperatorMap().Optional()),
	zod.F("length", numberOperatorMap().Optional()),
	zod.F("height", numberOperatorMap().Optional()),
	zod.F("width", numberOperatorMap().Optional()),
	zod.F("location_levels", zod.Object(zod.F("location_id", utils.StringOrStrings().Optional())).Optional()),
)

// AdminGetInventoryItemsParams mirrors AdminGetInventoryItemsParams.
var AdminGetInventoryItemsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(20), Offset: utils.Num(0)}).
	Merge(AdminGetInventoryItemsParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetInventoryItemsParamsFields)).Strict()

// AdminGetInventoryLocationLevelsParamsFields mirrors AdminGetInventoryLocationLevelsParamsFields.
var AdminGetInventoryLocationLevelsParamsFields = zod.Object(zod.F("location_id", utils.StringOrStrings().Optional()))

// AdminGetInventoryLocationLevelsParams mirrors AdminGetInventoryLocationLevelsParams.
var AdminGetInventoryLocationLevelsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(50), Offset: utils.Num(0)}).
	Merge(AdminGetInventoryLocationLevelsParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetInventoryLocationLevelsParamsFields))

// Middlewares mirrors adminInventoryRoutesMiddlewares (reads).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/inventory-items", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetInventoryItemsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/inventory-items/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/inventory-items/:id/location-levels", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetInventoryLocationLevelsParams, ListLocationLevelsConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("inventory_items", "inventory_items")}
