package products

import (
	"context"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// AdminCreateVariantInventoryItem mirrors AdminCreateVariantInventoryItem.
var AdminCreateVariantInventoryItem = zod.Object(
	zod.F("required_quantity", zod.Number()),
	zod.F("inventory_item_id", zod.String()),
)

// AdminUpdateVariantInventoryItem mirrors AdminUpdateVariantInventoryItem.
var AdminUpdateVariantInventoryItem = zod.Object(
	zod.F("required_quantity", zod.Number()),
)

// AdminBatchCreateVariantInventoryItem mirrors AdminBatchCreateVariantInventoryItem.
var AdminBatchCreateVariantInventoryItem = zod.Object(
	zod.F("required_quantity", zod.Number()),
	zod.F("inventory_item_id", zod.String()),
	zod.F("variant_id", zod.String()),
).Strict()

// AdminBatchUpdateVariantInventoryItem mirrors AdminBatchUpdateVariantInventoryItem.
var AdminBatchUpdateVariantInventoryItem = zod.Object(
	zod.F("required_quantity", zod.Number()),
	zod.F("inventory_item_id", zod.String()),
	zod.F("variant_id", zod.String()),
).Strict()

// AdminBatchDeleteVariantInventoryItem mirrors AdminBatchDeleteVariantInventoryItem.
var AdminBatchDeleteVariantInventoryItem = zod.Object(
	zod.F("inventory_item_id", zod.String()),
	zod.F("variant_id", zod.String()),
).Strict()

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// VariantInventoryMiddlewares mirrors the variant inventory item entries of
// adminProductRoutesMiddlewares.
var VariantInventoryMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/products/:id/variants/inventory-items/batch", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(utils.CreateBatchBody(AdminBatchCreateVariantInventoryItem, AdminBatchUpdateVariantInventoryItem, AdminBatchDeleteVariantInventoryItem))),
		httputils.ValidateAndTransformQuery(AdminGetProductVariantParams, RetrieveVariantConfig),
	)},
	{Methods: []string{"POST"}, Matcher: "/admin/products/:id/variants/:variant_id/inventory-items", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateVariantInventoryItem)),
		httputils.ValidateAndTransformQuery(AdminGetProductVariantParams, RetrieveVariantConfig),
	)},
	{Methods: []string{"POST"}, Matcher: "/admin/products/:id/variants/:variant_id/inventory-items/:inventory_item_id", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateVariantInventoryItem)),
		httputils.ValidateAndTransformQuery(AdminGetProductVariantParams, RetrieveVariantConfig),
	)},
	{Methods: []string{"DELETE"}, Matcher: "/admin/products/:id/variants/:variant_id/inventory-items/:inventory_item_id", Middlewares: mw(
		httputils.ValidateAndTransformQuery(AdminGetProductVariantParams, RetrieveVariantConfig),
	)},
}

// AdminBatchImageVariant mirrors AdminBatchImageVariant (and
// AdminBatchVariantImages, the same shape).
var AdminBatchImageVariant = zod.Object(
	zod.F("add", zod.Array(zod.String()).Optional()),
	zod.F("remove", zod.Array(zod.String()).Optional()),
)

func init() {
	AdminProductRoutesMiddlewares = append(AdminProductRoutesMiddlewares, VariantInventoryMiddlewares...)
	AdminProductRoutesMiddlewares = append(AdminProductRoutesMiddlewares,
		medusahttp.MiddlewareRoute{Methods: []string{"POST"}, Matcher: "/admin/products/:id/images/:image_id/variants/batch", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminBatchImageVariant)))},
		medusahttp.MiddlewareRoute{Methods: []string{"POST"}, Matcher: "/admin/products/:id/variants/:variant_id/images/batch", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminBatchImageVariant)))},
	)
}

// BatchImagesRoute builds the POST handler of the image batch routes.
func BatchImagesRoute(param string, run func(ctx context.Context, scope medusahttp.MedusaContainer, id any, add, remove []any) ([]any, []any, error)) medusahttp.RouteHandler {
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
		var add, remove []any
		if body, ok := req.ValidatedBody.(*dal.Record); ok {
			a, _ := body.Get("add")
			r, _ := body.Get("remove")
			add, _ = a.([]any)
			remove, _ = r.([]any)
		}
		added, removed, err := run(req.Ctx(), req.Scope, req.Params[param], add, remove)
		if err != nil {
			return err
		}
		out := dal.NewRecord()
		out.Set("added", added)
		out.Set("removed", removed)
		return res.Status(200).JSON(out)
	}
}

// RefetchVariant mirrors refetchVariant.
func RefetchVariant(ctx context.Context, variantID any, scope medusahttp.MedusaContainer, fields []string) (any, error) {
	f := dal.NewRecord()
	f.Set("id", variantID)
	v, err := httputils.RemoteQueryFirst(ctx, scope, "product_variant", f, RemapKeysForVariant(fields))
	if err != nil {
		return nil, err
	}
	return RemapVariantResponse(v), nil
}

// VariantInventoryLink mirrors the { product: { variant_id }, inventory:
// { inventory_item_id }, data? } link definition.
func VariantInventoryLink(variantID, inventoryItemID any, data *dal.Record) *dal.Record {
	p := dal.NewRecord()
	p.Set("variant_id", variantID)
	i := dal.NewRecord()
	i.Set("inventory_item_id", inventoryItemID)
	l := dal.NewRecord()
	l.Set(utilsmodulessdk.PRODUCT, p)
	l.Set(utilsmodulessdk.INVENTORY, i)
	if data != nil {
		l.Set("data", data)
	}
	return l
}

// BuildBatchVariantInventoryData mirrors buildBatchVariantInventoryData.
func BuildBatchVariantInventoryData(inputs any) []*dal.Record {
	list, _ := inputs.([]any)
	out := []*dal.Record{}
	for _, x := range list {
		in, _ := x.(*dal.Record)
		if in == nil {
			continue
		}
		v, _ := in.Get("variant_id")
		i, _ := in.Get("inventory_item_id")
		var data *dal.Record
		if q, ok := in.Get("required_quantity"); ok {
			data = dal.NewRecord()
			data.Set("required_quantity", q)
		}
		out = append(out, VariantInventoryLink(v, i, data))
	}
	return out
}
