package product

import (
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

func toStringJS(v any) string {
	switch t := v.(type) {
	case bool:
		if t {
			return "true"
		}
		return "false"
	case string:
		return t
	}
	return fmt.Sprint(v)
}

// booleanString mirrors booleanString().
func booleanString() *zod.Schema {
	return zod.Union(zod.Boolean(), zod.String()).
		Refine(func(v any) bool {
			s := strings.ToLower(toStringJS(v))
			return s == "true" || s == "false"
		}, "Invalid input").
		Transform(func(v any) (any, error) { return strings.ToLower(toStringJS(v)) == "true", nil })
}

// numericString mirrors numericString(): numbers become strings.
func numericString() *zod.Schema {
	return zod.Number().Transform(func(v any) (any, error) {
		if f, ok := v.(float64); ok {
			return totals.JSNumberToString(f), nil
		}
		return v, nil
	})
}

var idAssociation = zod.Object(zod.F("id", zod.String()))

func statusEnum() *zod.Schema { return zod.Enum("draft", "proposed", "published", "rejected") }

func metadata() *zod.Schema { return zod.Record(zod.String(), zod.Unknown()).Nullish() }

var createVariantPrice = zod.Object(
	zod.F("currency_code", zod.String()),
	zod.F("amount", zod.Number()),
	zod.F("min_quantity", zod.Number().Nullish()),
	zod.F("max_quantity", zod.Number().Nullish()),
	zod.F("rules", zod.Record(zod.String(), zod.String()).Optional()),
)

var createProductOption = zod.Object(
	zod.F("id", zod.String().Optional()),
	zod.F("title", zod.String().Optional()),
	zod.F("values", zod.Array(zod.String()).Optional()),
	zod.F("is_exclusive", booleanString().Optional()),
)

var createProductVariant = zod.Object(
	zod.F("title", zod.String()),
	zod.F("sku", zod.String().Nullish()),
	zod.F("ean", zod.String().Nullish()),
	zod.F("upc", zod.String().Nullish()),
	zod.F("barcode", zod.String().Nullish()),
	zod.F("hs_code", zod.String().Nullish()),
	zod.F("mid_code", zod.String().Nullish()),
	zod.F("allow_backorder", booleanString().Optional().Default(false)),
	zod.F("manage_inventory", booleanString().Optional().Default(true)),
	zod.F("variant_rank", zod.Number().Optional()),
	zod.F("weight", zod.Number().Nullish()),
	zod.F("length", zod.Number().Nullish()),
	zod.F("height", zod.Number().Nullish()),
	zod.F("width", zod.Number().Nullish()),
	zod.F("origin_country", zod.String().Nullish()),
	zod.F("material", zod.String().Nullish()),
	zod.F("metadata", metadata()),
	zod.F("prices", zod.Array(createVariantPrice)),
	zod.F("options", zod.Record(zod.String(), zod.String()).Optional()),
	zod.F("inventory_items", zod.Array(zod.Object(zod.F("inventory_item_id", zod.String()), zod.F("required_quantity", zod.Number()))).Optional()),
).Strict()

// CreateProduct mirrors productValidators.CreateProduct.
var CreateProduct = zod.Object(
	zod.F("title", zod.String()),
	zod.F("subtitle", zod.String().Nullish()),
	zod.F("description", zod.String().Nullish()),
	zod.F("is_giftcard", booleanString().Optional().Default(false)),
	zod.F("discountable", booleanString().Optional().Default(true)),
	zod.F("images", zod.Array(zod.Object(zod.F("url", zod.String()))).Optional()),
	zod.F("thumbnail", zod.String().Nullish()),
	zod.F("handle", zod.String().Optional()),
	zod.F("status", statusEnum().Optional().Default("draft")),
	zod.F("external_id", zod.String().Nullish()),
	zod.F("type_id", zod.String().Nullish()),
	zod.F("collection_id", zod.String().Nullish()),
	zod.F("categories", zod.Array(idAssociation).Optional()),
	zod.F("tags", zod.Array(idAssociation).Optional()),
	zod.F("options", zod.Array(createProductOption).Optional()),
	zod.F("variants", zod.Array(createProductVariant).Optional()),
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
	zod.F("metadata", metadata()),
).Strict()

var updateVariantPrice = zod.Object(
	zod.F("id", zod.String().Optional()),
	zod.F("currency_code", zod.String().Optional()),
	zod.F("amount", zod.Number().Optional()),
	zod.F("min_quantity", zod.Number().Nullish()),
	zod.F("max_quantity", zod.Number().Nullish()),
	zod.F("rules", zod.Record(zod.String(), zod.String()).Optional()),
)

var updateProductVariant = zod.Object(
	zod.F("id", zod.String().Optional()),
	zod.F("title", zod.String().Optional()),
	zod.F("prices", zod.Array(updateVariantPrice).Optional()),
	zod.F("sku", zod.String().Nullish()),
	zod.F("ean", zod.String().Nullish()),
	zod.F("upc", zod.String().Nullish()),
	zod.F("barcode", zod.String().Nullish()),
	zod.F("hs_code", zod.String().Nullish()),
	zod.F("mid_code", zod.String().Nullish()),
	zod.F("allow_backorder", booleanString().Optional()),
	zod.F("manage_inventory", booleanString().Optional()),
	zod.F("variant_rank", zod.Number().Optional()),
	zod.F("weight", numericString().Nullish()),
	zod.F("length", numericString().Nullish()),
	zod.F("height", numericString().Nullish()),
	zod.F("width", numericString().Nullish()),
	zod.F("origin_country", zod.String().Nullish()),
	zod.F("material", zod.String().Nullish()),
	zod.F("metadata", metadata()),
	zod.F("options", zod.Record(zod.String(), zod.String()).Optional()),
).Strict()

// UpdateProduct mirrors productValidators.UpdateProduct.
var UpdateProduct = zod.Object(
	zod.F("id", zod.String()),
	zod.F("title", zod.String().Optional()),
	zod.F("discountable", booleanString().Optional()),
	zod.F("is_giftcard", booleanString().Optional()),
	zod.F("options", zod.Array(createProductOption).Optional()),
	zod.F("variants", zod.Array(updateProductVariant).Optional()),
	zod.F("status", statusEnum().Optional()),
	zod.F("subtitle", zod.String().Nullish()),
	zod.F("description", zod.String().Nullish()),
	zod.F("images", zod.Array(zod.Object(zod.F("url", zod.String()))).Optional()),
	zod.F("thumbnail", zod.String().Nullish()),
	zod.F("handle", zod.String().Nullish()),
	zod.F("type_id", zod.String().Nullish()),
	zod.F("external_id", zod.String().Nullish()),
	zod.F("collection_id", zod.String().Nullish()),
	zod.F("categories", zod.Array(idAssociation).Optional()),
	zod.F("tags", zod.Array(idAssociation).Optional()),
	zod.F("sales_channels", zod.Array(zod.Object(zod.F("id", zod.String()))).Optional()),
	zod.F("shipping_profile_id", zod.String().Nullish()),
	zod.F("weight", numericString().Nullish()),
	zod.F("length", numericString().Nullish()),
	zod.F("height", numericString().Nullish()),
	zod.F("width", numericString().Nullish()),
	zod.F("hs_code", zod.String().Nullish()),
	zod.F("mid_code", zod.String().Nullish()),
	zod.F("origin_country", zod.String().Nullish()),
	zod.F("material", zod.String().Nullish()),
	zod.F("metadata", metadata()),
).Strict()
