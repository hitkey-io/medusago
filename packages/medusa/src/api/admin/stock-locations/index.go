// Package stocklocations mirrors api/admin/stock-locations.
package stocklocations

import (
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/stock-location/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
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
func trim(v any) any {
	if str, ok := v.(string); ok {
		return strings.TrimSpace(str)
	}
	return v
}

// AdminUpsertStockLocationAddress mirrors AdminUpsertStockLocationAddress.
var AdminUpsertStockLocationAddress = zod.Object(
	zod.F("address_1", zod.String()),
	zod.F("address_2", zod.String().Nullish()),
	zod.F("company", zod.String().Nullish()),
	zod.F("city", zod.String().Nullish()),
	zod.F("country_code", zod.String()),
	zod.F("phone", zod.String().Nullish()),
	zod.F("postal_code", zod.String().Nullish()),
	zod.F("province", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

// AdminCreateStockLocation mirrors AdminCreateStockLocation.
var AdminCreateStockLocation = zod.Object(
	zod.F("name", zod.Preprocess(trim, zod.String())),
	zod.F("address", AdminUpsertStockLocationAddress.Optional()),
	zod.F("address_id", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

// AdminUpdateStockLocation mirrors AdminUpdateStockLocation.
var AdminUpdateStockLocation = zod.Object(
	zod.F("name", zod.Preprocess(trim, zod.String().Optional()).Optional()),
	zod.F("address", AdminUpsertStockLocationAddress.Optional()),
	zod.F("address_id", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/stock-locations", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateStockLocation)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"GET"}, Matcher: "/admin/stock-locations", Middlewares: []medusahttp.MiddlewareFunction{
		httputils.ValidateAndTransformQuery(AdminGetStockLocationsParams, ListConfig),
		httputils.MaybeApplyLinkFilter(httputils.LinkFilterOptions{EntryPoint: "sales_channel_location", ResourceID: "stock_location_id", FilterableField: "sales_channel_id"}),
	}},
	{Methods: []string{"GET"}, Matcher: "/admin/stock-locations/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/stock-locations/:id", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateStockLocation)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/stock-locations/:id/sales-channels", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(utils.CreateLinkBody())), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/stock-locations/:id/fulfillment-providers", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(utils.CreateLinkBody())), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("stock_locations", "stock_locations"), POST: POST}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	result, err := workflows.CreateStockLocationsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{body})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	return Respond(req, res, id)
}

// Respond refetches the stock location and writes { stock_location }.
func Respond(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id any) error {
	f := dal.NewRecord()
	f.Set("id", id)
	v, err := httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "stock_location", f, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if v != nil {
		out.Set("stock_location", v)
	}
	return res.Status(200).JSON(out)
}

// Strings converts a validated string array.
func Strings(v any) []string {
	l, _ := v.([]any)
	if l == nil {
		return nil
	}
	out := make([]string, len(l))
	for i, x := range l {
		out[i] = fmt.Sprint(x)
	}
	return out
}
