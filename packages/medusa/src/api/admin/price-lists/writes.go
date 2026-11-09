package pricelists

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/price-list/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// AdminCreatePriceListPrice mirrors AdminCreatePriceListPrice.
var AdminCreatePriceListPrice = zod.Object(
	zod.F("currency_code", zod.String()),
	zod.F("amount", zod.Number()),
	zod.F("variant_id", zod.String()),
	zod.F("min_quantity", zod.Number().Nullish()),
	zod.F("max_quantity", zod.Number().Nullish()),
	zod.F("rules", zod.Record(zod.String(), zod.String()).Optional()),
)

// AdminUpdatePriceListPrice mirrors AdminUpdatePriceListPrice.
var AdminUpdatePriceListPrice = zod.Object(
	zod.F("id", zod.String()),
	zod.F("currency_code", zod.String().Optional()),
	zod.F("amount", zod.Number().Optional()),
	zod.F("variant_id", zod.String()),
	zod.F("min_quantity", zod.Number().Nullish()),
	zod.F("max_quantity", zod.Number().Nullish()),
	zod.F("rules", zod.Record(zod.String(), zod.String()).Optional()),
)

// AdminCreatePriceList mirrors AdminCreatePriceList.
var AdminCreatePriceList = zod.Object(
	zod.F("title", zod.String()),
	zod.F("description", zod.String()),
	zod.F("starts_at", zod.String().Nullish()),
	zod.F("ends_at", zod.String().Nullish()),
	zod.F("status", zod.Enum("active", "draft").Optional()),
	zod.F("type", zod.Enum("sale", "override").Optional()),
	zod.F("rules", zod.Record(zod.String(), zod.Array(zod.String())).Optional()),
	zod.F("prices", zod.Array(AdminCreatePriceListPrice).Optional()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

// AdminUpdatePriceList mirrors AdminUpdatePriceList.
var AdminUpdatePriceList = zod.Object(
	zod.F("title", zod.String().Optional()),
	zod.F("description", zod.String().Nullish()),
	zod.F("starts_at", zod.String().Nullish()),
	zod.F("ends_at", zod.String().Nullish()),
	zod.F("status", zod.Enum("active", "draft").Optional()),
	zod.F("type", zod.Enum("sale", "override").Optional()),
	zod.F("rules", zod.Record(zod.String(), zod.Array(zod.String())).Optional()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

// AdminRemoveProductsPriceList mirrors AdminRemoveProductsPriceList.
var AdminRemoveProductsPriceList = zod.Object(
	zod.F("remove", zod.Array(zod.String()).Optional()),
)

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// WriteMiddlewares mirrors the write entries of
// adminPriceListsRoutesMiddlewares (authorize is RBAC and not ported).
var WriteMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/price-lists", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(AdminCreatePriceList)),
		httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig),
	)},
	{Methods: []string{"POST"}, Matcher: "/admin/price-lists/:id", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdatePriceList)),
		httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig),
	)},
	{Methods: []string{"POST"}, Matcher: "/admin/price-lists/:id/products", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(AdminRemoveProductsPriceList)),
		httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), ListConfig),
	)},
	{Methods: []string{"GET"}, Matcher: "/admin/price-lists/:id/prices", Middlewares: mw(
		httputils.ValidateAndTransformQuery(AdminGetPriceListPricesParams, ListPriceConfig),
	)},
	{Methods: []string{"POST"}, Matcher: "/admin/price-lists/:id/prices/batch", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(utils.CreateBatchBody(AdminCreatePriceListPrice, AdminUpdatePriceListPrice, nil))),
		httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), ListPriceConfig),
	)},
}

func init() {
	Middlewares = append(Middlewares, WriteMiddlewares...)
}

// FetchPriceList mirrors fetchPriceList.
func FetchPriceList(ctx context.Context, scope medusahttp.MedusaContainer, id string, fields []string) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	pl, err := httputils.RemoteQueryFirst(ctx, scope, "price_lists", f, fields)
	if err != nil {
		return nil, err
	}
	if pl == nil {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Price list with id: %s was not found", id))
	}
	return TransformPriceList(pl), nil
}

// ListPrices mirrors listPrices.
func ListPrices(ctx context.Context, scope medusahttp.MedusaContainer, ids []any, fields []string) ([]any, error) {
	if ids == nil {
		ids = []any{}
	}
	f := dal.NewRecord()
	f.Set("id", ids)
	vars := dal.NewRecord()
	vars.Set("filters", f)
	r, err := httputils.RemoteQueryList(ctx, scope, "price", vars, fields)
	if err != nil {
		return nil, err
	}
	if r.Data == nil {
		return []any{}, nil
	}
	return r.Data, nil
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	result, err := workflows.CreatePriceListsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{body})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	pl, err := FetchPriceList(req.Ctx(), req.Scope, fmt.Sprint(id), req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("price_list", pl)
	return res.Status(200).JSON(out)
}
