package utils

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

// PrepareLineItemDataInput mirrors PrepareLineItemDataInput.
type PrepareLineItemDataInput struct {
	Item           *dal.Record
	IsCustomPrice  bool
	Variant        *dal.Record
	TaxLines       []*dal.Record
	Adjustments    []*dal.Record
	CartID         string
	UnitPrice      any
	IsTaxInclusive bool
}

// PrepareLineItemData mirrors prepareLineItemData.
func PrepareLineItemData(data PrepareLineItemDataInput) (*dal.Record, error) {
	item, variant := data.Item, data.Variant
	product := Rec(variant, "product")
	if variant != nil && product == nil {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Variant does not have a product")
	}
	if item != nil && totals.Lte(Get(item, "quantity"), 0) {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Item quantity must be greater than 0")
	}
	compareAt := Get(item, "compare_at_unit_price")
	cp := Rec(variant, "calculated_price")
	isSale := Get(Rec(cp, "calculated_price"), "price_list_type") == "sale"
	if !IsPresent(compareAt) && isSale && !totals.Eq(Get(cp, "original_amount"), Get(cp, "calculated_amount")) {
		compareAt = Get(cp, "original_amount")
	}
	hasShippingProfile := Has(Rec(product, "shipping_profile"), "id")
	someInventoryRequiresShipping := false
	for _, ii := range Records(Get(variant, "inventory_items")) {
		if Truthy(Get(Rec(ii, "inventory"), "requires_shipping")) {
			someInventoryRequiresShipping = true
		}
	}
	var requiresShipping any = hasShippingProfile || someInventoryRequiresShipping
	if Has(item, "requires_shipping") {
		requiresShipping = Get(item, "requires_shipping")
	}
	pick := func(itemKey string, fallback any) any { return Coalesce(Get(item, itemKey), fallback) }
	li := dal.NewRecord()
	li.Set("quantity", Get(item, "quantity"))
	li.Set("title", pick("title", Get(product, "title")))
	li.Set("subtitle", pick("subtitle", Get(variant, "title")))
	li.Set("thumbnail", Coalesce(Get(item, "thumbnail"), Get(variant, "thumbnail"), Get(product, "thumbnail")))
	li.Set("product_id", Coalesce(Get(product, "id"), Get(item, "product_id")))
	li.Set("product_title", pick("product_title", Get(product, "title")))
	li.Set("product_description", pick("product_description", Get(product, "description")))
	li.Set("product_subtitle", pick("product_subtitle", Get(product, "subtitle")))
	li.Set("product_type", Coalesce(Get(item, "product_type"), Get(Rec(product, "type"), "value")))
	li.Set("product_type_id", Coalesce(Get(item, "product_type_id"), Get(Rec(product, "type"), "id")))
	li.Set("product_collection", Coalesce(Get(item, "product_collection"), Get(Rec(product, "collection"), "title")))
	li.Set("product_handle", pick("product_handle", Get(product, "handle")))
	li.Set("variant_id", Get(variant, "id"))
	li.Set("variant_sku", pick("variant_sku", Get(variant, "sku")))
	li.Set("variant_barcode", pick("variant_barcode", Get(variant, "barcode")))
	li.Set("variant_title", pick("variant_title", Get(variant, "title")))
	li.Set("variant_option_values", Get(item, "variant_option_values"))
	// undefined upstream (dropped by the JSON step boundary) -> the column default applies
	if d := pick("is_discountable", Get(product, "discountable")); d != nil {
		li.Set("is_discountable", d)
	}
	li.Set("is_giftcard", Coalesce(Get(product, "is_giftcard"), false))
	li.Set("requires_shipping", requiresShipping)
	li.Set("unit_price", data.UnitPrice)
	li.Set("compare_at_unit_price", compareAt)
	li.Set("is_tax_inclusive", data.IsTaxInclusive)
	li.Set("metadata", Coalesce(Get(item, "metadata"), dal.NewRecord()))
	if data.IsCustomPrice {
		li.Set("is_custom_price", true)
	}
	if data.TaxLines != nil {
		li.Set("tax_lines", PrepareTaxLinesData(data.TaxLines))
	}
	if data.Adjustments != nil {
		li.Set("adjustments", PrepareAdjustmentsData(data.Adjustments))
	}
	if data.CartID != "" {
		li.Set("cart_id", data.CartID)
	}
	return li, nil
}

// PrepareTaxLinesData mirrors prepareTaxLinesData.
func PrepareTaxLinesData(data []*dal.Record) []any {
	out := make([]any, len(data))
	for i, d := range data {
		r := dal.NewRecord()
		r.Set("description", Get(d, "description"))
		r.Set("tax_rate_id", Get(d, "tax_rate_id"))
		r.Set("code", Get(d, "code"))
		r.Set("rate", Get(d, "rate"))
		r.Set("provider_id", Get(d, "provider_id"))
		r.Set("metadata", Get(d, "metadata"))
		r.Set("data", Get(d, "data"))
		out[i] = r
	}
	return out
}

// PrepareAdjustmentsData mirrors prepareAdjustmentsData.
func PrepareAdjustmentsData(data []*dal.Record) []any {
	out := make([]any, len(data))
	for i, d := range data {
		r := dal.NewRecord()
		for _, k := range []string{"code", "amount", "description", "promotion_id", "provider_id", "is_tax_inclusive"} {
			r.Set(k, Get(d, k))
		}
		out[i] = r
	}
	return out
}
