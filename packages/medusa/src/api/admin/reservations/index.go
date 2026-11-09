// Package reservations mirrors api/admin/reservations.
package reservations

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	inventoryitems "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/inventory-items"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// DefaultAdminReservationFields mirrors defaultAdminReservationFields.
var DefaultAdminReservationFields = func() []string {
	out := []string{"id", "location_id", "inventory_item_id", "quantity", "line_item_id", "description", "metadata", "created_at", "updated_at"}
	for _, f := range inventoryitems.DefaultAdminInventoryItemFields {
		out = append(out, "inventory_item."+f)
	}
	return out
}()

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultAdminReservationFields, Entity: "reservation_item"}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultAdminReservationFields, IsList: true, Entity: "reservation_item"}
)

// AdminGetReservationsParams mirrors AdminGetReservationsParams.
var AdminGetReservationsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(20), Offset: utils.Num(0)}).Merge(zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("location_id", utils.StringOrStrings().Optional()),
	zod.F("inventory_item_id", utils.StringOrStrings().Optional()),
	zod.F("line_item_id", utils.StringOrStrings().Optional()),
	zod.F("created_by", utils.StringOrStrings().Optional()),
	zod.F("description", zod.Union(zod.String(), utils.CreateOperatorMap(nil, nil)).Optional()),
	zod.F("quantity", utils.CreateOperatorMap(zod.Number(), utils.ParseFloat).Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
))

// Middlewares mirrors adminReservationRoutesMiddlewares (reads).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/reservations", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetReservationsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/reservations/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("reservation", "reservations")}
