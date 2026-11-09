package inventoryitems

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/inventory/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

func qty() *zod.Schema { return zod.Number().Min(0).Optional() }

// AdminCreateInventoryLocationLevel mirrors AdminCreateInventoryLocationLevel.
var AdminCreateInventoryLocationLevel = zod.Object(
	zod.F("location_id", zod.String()),
	zod.F("stocked_quantity", qty()),
	zod.F("incoming_quantity", qty()),
).Strict()

// AdminUpdateInventoryLocationLevelBatch mirrors AdminUpdateInventoryLocationLevelBatch.
var AdminUpdateInventoryLocationLevelBatch = zod.Object(
	zod.F("id", zod.String().Optional()),
	zod.F("location_id", zod.String()),
	zod.F("stocked_quantity", qty()),
	zod.F("incoming_quantity", qty()),
).Strict()

// AdminBatchInventoryItemLocationsLevel mirrors AdminBatchInventoryItemLocationsLevel.
var AdminBatchInventoryItemLocationsLevel = zod.Object(
	zod.F("create", zod.Array(AdminCreateInventoryLocationLevel).Optional()),
	zod.F("update", zod.Array(AdminUpdateInventoryLocationLevelBatch).Optional()),
	zod.F("delete", zod.Array(zod.String()).Optional()),
	zod.F("force", zod.Boolean().Optional()),
)

// AdminUpdateInventoryLocationLevel mirrors AdminUpdateInventoryLocationLevel.
var AdminUpdateInventoryLocationLevel = zod.Object(
	zod.F("stocked_quantity", qty()),
	zod.F("incoming_quantity", qty()),
).Strict()

func itemFields() []zod.Field {
	return []zod.Field{
		zod.F("sku", zod.String().Nullish()),
		zod.F("hs_code", zod.String().Nullish()),
		zod.F("weight", zod.Number().Nullish()),
		zod.F("length", zod.Number().Nullish()),
		zod.F("height", zod.Number().Nullish()),
		zod.F("width", zod.Number().Nullish()),
		zod.F("origin_country", zod.String().Nullish()),
		zod.F("mid_code", zod.String().Nullish()),
		zod.F("material", zod.String().Nullish()),
		zod.F("unit_of_measure", zod.String().Nullish()),
		zod.F("title", zod.String().Nullish()),
		zod.F("description", zod.String().Nullish()),
		zod.F("requires_shipping", zod.Boolean().Optional()),
		zod.F("thumbnail", zod.String().Nullish()),
		zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
	}
}

// AdminCreateInventoryItem mirrors AdminCreateInventoryItem.
var AdminCreateInventoryItem = zod.Object(append(itemFields(), zod.F("location_levels", zod.Array(AdminCreateInventoryLocationLevel).Optional()))...).Strict()

// AdminUpdateInventoryItem mirrors AdminUpdateInventoryItem.
var AdminUpdateInventoryItem = zod.Object(itemFields()...).Strict()

var adminBatchInventoryLocationLevel = zod.Object(
	zod.F("inventory_item_id", zod.String()),
	zod.F("location_id", zod.String()),
	zod.F("stocked_quantity", qty()),
	zod.F("incoming_quantity", qty()),
)

// AdminBatchInventoryItemLevels mirrors AdminBatchInventoryItemLevels.
var AdminBatchInventoryItemLevels = zod.Object(
	zod.F("create", zod.Array(adminBatchInventoryLocationLevel).Optional()),
	zod.F("update", zod.Array(adminBatchInventoryLocationLevel).Optional()),
	zod.F("delete", zod.Array(zod.String()).Optional()),
	zod.F("force", zod.Boolean().Optional()),
).Strict()

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// WriteMiddlewares mirrors the write routes of adminInventoryRoutesMiddlewares.
var WriteMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/inventory-items", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateInventoryItem)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/inventory-items/location-levels/batch", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminBatchInventoryItemLevels)))},
	{Methods: []string{"POST"}, Matcher: "/admin/inventory-items/:id", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateInventoryItem)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/inventory-items/:id/location-levels", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateInventoryLocationLevel)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/inventory-items/:id/location-levels/batch", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminBatchInventoryItemLocationsLevel)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveLocationLevelConfig))},
	{Methods: []string{"DELETE"}, Matcher: "/admin/inventory-items/:id/location-levels/:location_id", Middlewares: mw(httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/inventory-items/:id/location-levels/:location_id", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateInventoryLocationLevel)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
}

func init() { Middlewares = append(Middlewares, WriteMiddlewares...) }

// Refetch mirrors refetchInventoryItem.
func Refetch(req *medusahttp.MedusaRequest, id any) (any, error) {
	filters := dal.NewRecord()
	filters.Set("id", id)
	vars := dal.NewRecord()
	vars.Set("filters", filters)
	vars.Set("skip", float64(0))
	vars.Set("take", float64(1))
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "inventory_item", vars, req.QueryConfig.Fields)
	if err != nil || len(r.Data) == 0 {
		return nil, err
	}
	return r.Data[0], nil
}

// Respond refetches and writes { inventory_item } with status 200.
func Respond(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id any) error {
	v, err := Refetch(req, id)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if v != nil {
		out.Set("inventory_item", v)
	}
	return res.Status(200).JSON(out)
}

// Records converts a validated array of objects.
func Records(v any) []*dal.Record {
	l, _ := v.([]any)
	var out []*dal.Record
	for _, x := range l {
		if r, ok := x.(*dal.Record); ok {
			out = append(out, r)
		}
	}
	return out
}

// Anys converts a validated array.
func Anys(v any) []any {
	l, _ := v.([]any)
	return l
}

// BatchResponse writes { created, updated, deleted }.
func BatchResponse(res *medusahttp.MedusaResponse, r workflows.BatchInventoryItemLevelsResult, status bool) error {
	out := dal.NewRecord()
	out.Set("created", r.Created)
	out.Set("updated", r.Updated)
	if r.Deleted == nil {
		out.Set("deleted", []any{})
	} else {
		out.Set("deleted", r.Deleted)
	}
	if status {
		return res.Status(200).JSON(out)
	}
	return res.JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	result, err := workflows.CreateInventoryItemsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{body})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	return Respond(req, res, id)
}

var _ = fmt.Sprint
