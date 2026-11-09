// Package products mirrors api/admin/products.
package products

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	productvariants "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-variants"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

func operatorMap() *zod.Schema { return utils.CreateOperatorMap(nil, nil) }

// AdminGetProductParams mirrors AdminGetProductParams.
var AdminGetProductParams = utils.CreateSelectParams()

// AdminGetProductsParamsDirectFields mirrors AdminGetProductsParamsDirectFields.
var AdminGetProductsParamsDirectFields = zod.Object(
	zod.F("variants", productvariants.AdminGetProductVariantsParamsFields.Omit("q").
		Merge(commonvalidators.ApplyAndAndOrOperators(productvariants.AdminGetProductVariantsParamsFields.Omit("q"))).Optional()),
	zod.F("status", commonvalidators.ProductStatusEnum().Array().Optional()),
)

// AdminGetProductsParams mirrors AdminGetProductsParams (a transform: not
// strict).
var AdminGetProductsParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(50)}).
	Merge(AdminGetProductsParamsDirectFields).
	Merge(zod.Object(zod.F("price_list_id", zod.String().Array().Optional())).
		Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetProductsParamsDirectFields)).
		Merge(commonvalidators.GetProductsParams(operatorMap))).
	Transform(commonvalidators.TransformProductParams)

// AdminCreateVariantPrice mirrors AdminCreateVariantPrice.
var AdminCreateVariantPrice = zod.Object(
	zod.F("currency_code", zod.String()),
	zod.F("amount", zod.Number()),
	zod.F("min_quantity", zod.Number().Nullish()),
	zod.F("max_quantity", zod.Number().Nullish()),
	zod.F("rules", zod.Record(zod.String(), zod.String()).Optional()),
)

// CreateProductVariant mirrors CreateProductVariant.
var CreateProductVariant = zod.Object(
	zod.F("title", zod.String()),
	zod.F("sku", zod.String().Nullish()),
	zod.F("ean", zod.String().Nullish()),
	zod.F("upc", zod.String().Nullish()),
	zod.F("barcode", zod.String().Nullish()),
	zod.F("hs_code", zod.String().Nullish()),
	zod.F("mid_code", zod.String().Nullish()),
	zod.F("allow_backorder", commonvalidators.BooleanString().Optional().Default(false)),
	zod.F("manage_inventory", commonvalidators.BooleanString().Optional().Default(true)),
	zod.F("variant_rank", zod.Number().Optional()),
	zod.F("weight", zod.Number().Nullish()),
	zod.F("length", zod.Number().Nullish()),
	zod.F("height", zod.Number().Nullish()),
	zod.F("width", zod.Number().Nullish()),
	zod.F("origin_country", zod.String().Nullish()),
	zod.F("material", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
	zod.F("prices", zod.Array(AdminCreateVariantPrice)),
	zod.F("options", zod.Record(zod.String(), zod.String()).Optional()),
	zod.F("inventory_items", zod.Array(zod.Object(
		zod.F("inventory_item_id", zod.String()),
		zod.F("required_quantity", zod.Number()),
	)).Optional()),
).Strict()

// AdminCreateProductOption mirrors AdminCreateProductOption.
var AdminCreateProductOption = zod.Object(
	zod.F("title", zod.String()),
	zod.F("values", zod.Array(zod.String())),
	zod.F("ranks", zod.Record(zod.String(), zod.Number()).Optional()),
	zod.F("is_exclusive", zod.Boolean().Optional()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
).Strict()

// IdAssociation mirrors IdAssociation.
var IdAssociation = zod.Object(zod.F("id", zod.String()))

// CreateProduct mirrors CreateProduct.
var CreateProduct = zod.Object(
	zod.F("title", zod.String()),
	zod.F("subtitle", zod.String().Nullish()),
	zod.F("description", zod.String().Nullish()),
	zod.F("is_giftcard", commonvalidators.BooleanString().Optional().Default(false)),
	zod.F("discountable", commonvalidators.BooleanString().Optional().Default(true)),
	zod.F("images", zod.Array(zod.Object(zod.F("url", zod.String()))).Optional()),
	zod.F("thumbnail", zod.String().Nullish()),
	zod.F("handle", zod.String().Optional()),
	zod.F("status", commonvalidators.ProductStatusEnum().Nullish().Default("draft")),
	zod.F("external_id", zod.String().Nullish()),
	zod.F("type_id", zod.String().Nullish()),
	zod.F("collection_id", zod.String().Nullish()),
	zod.F("categories", zod.Array(IdAssociation).Optional()),
	zod.F("tags", zod.Array(IdAssociation).Optional()),
	zod.F("options", zod.Array(zod.Union(
		AdminCreateProductOption,
		zod.Object(zod.F("id", zod.String()), zod.F("value_ids", zod.Array(zod.String()).Optional())),
	)).Optional()),
	zod.F("variants", zod.Array(CreateProductVariant).Optional()),
	zod.F("sales_channels", zod.Array(zod.Object(zod.F("id", zod.String()))).Optional()),
	zod.F("shipping_profile_id", zod.String().Optional()),
	zod.F("weight", zod.Number().Nullish()),
	zod.F("length", zod.Number().Nullish()),
	zod.F("height", zod.Number().Nullish()),
	zod.F("width", zod.Number().Nullish()),
	zod.F("hs_code", zod.String().Nullish()),
	zod.F("mid_code", zod.String().Nullish()),
	zod.F("origin_country", zod.String().Nullish()),
	zod.F("material", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
).Strict()

// AdminCreateProduct mirrors AdminCreateProduct.
var AdminCreateProduct = utils.WithAdditionalData(CreateProduct)
