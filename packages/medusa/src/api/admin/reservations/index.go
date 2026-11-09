// Package reservations mirrors api/admin/reservations.
package reservations

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/reservation/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
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

// AdminCreateReservation mirrors AdminCreateReservation.
var AdminCreateReservation = zod.Object(
	zod.F("line_item_id", zod.String().Nullish()),
	zod.F("location_id", zod.String()),
	zod.F("inventory_item_id", zod.String()),
	zod.F("quantity", zod.Number()),
	zod.F("description", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
).Strict()

// AdminUpdateReservation mirrors AdminUpdateReservation.
var AdminUpdateReservation = zod.Object(
	zod.F("location_id", zod.String().Optional()),
	zod.F("quantity", zod.Number().Optional()),
	zod.F("description", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
).Strict()

// Middlewares mirrors adminReservationRoutesMiddlewares (authorize is RBAC
// and not ported).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/reservations", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetReservationsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/reservations/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/reservations", Middlewares: []medusahttp.MiddlewareFunction{
		httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateReservation)),
		httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig),
	}},
	{Methods: []string{"POST"}, Matcher: "/admin/reservations/:id", Middlewares: []medusahttp.MiddlewareFunction{
		httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateReservation)),
		httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig),
	}},
}

// RefetchReservation mirrors refetchReservation.
func RefetchReservation(ctx context.Context, id any, scope medusahttp.MedusaContainer, fields []string) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	return httputils.RemoteQueryFirst(ctx, scope, "reservation", f, fields)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	result, err := workflows.CreateReservationsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{body})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	reservation, err := RefetchReservation(req.Ctx(), id, req.Scope, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("reservation", reservation)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("reservation", "reservations"), POST: POST}
