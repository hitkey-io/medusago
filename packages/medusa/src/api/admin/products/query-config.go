package products

import (
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	productvariants "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-variants"
)

func without(list []string, x string) []string {
	var out []string
	for _, f := range list {
		if f != x {
			out = append(out, f)
		}
	}
	return out
}

// DefaultAdminProductsVariantFields mirrors defaultAdminProductsVariantFields.
var DefaultAdminProductsVariantFields = without(productvariants.DefaultAdminProductVariantFields, "*product")

// ListVariantConfig mirrors listVariantConfig.
var ListVariantConfig = httputils.QueryConfig{Defaults: DefaultAdminProductsVariantFields, DefaultLimit: 50, IsList: true, Entity: "product_variant"}

// DefaultAdminProductFields mirrors defaultAdminProductFields.
var DefaultAdminProductFields = []string{
	"id", "title", "subtitle", "status", "external_id", "description", "handle", "is_giftcard", "discountable",
	"thumbnail", "collection_id", "type_id", "weight", "length", "height", "width", "hs_code", "origin_country",
	"mid_code", "material", "created_at", "updated_at", "deleted_at", "metadata", "*type", "*collection",
	"*options", "*options.values", "*tags", "*images", "*variants", "*variants.prices",
	"variants.prices.price_rules.value", "variants.prices.price_rules.attribute", "*variants.options",
	"*variants.images", "*sales_channels",
}

// RetrieveProductQueryConfig / ListProductQueryConfig mirror the configs.
var (
	RetrieveProductQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminProductFields, Entity: "product"}
	ListProductQueryConfig     = httputils.QueryConfig{Defaults: DefaultAdminProductFields, DefaultLimit: 50, IsList: true, Entity: "product"}
)
