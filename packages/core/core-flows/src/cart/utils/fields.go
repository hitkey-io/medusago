package utils

// FieldsForPricingContext mirrors fieldsForPricingContext.
var FieldsForPricingContext = []string{"id", "sales_channel_id", "currency_code", "region_id", "shipping_address.city", "shipping_address.country_code", "shipping_address.province", "shipping_address.postal_code", "item_total", "total", "locale", "customer.id", "email", "customer.groups.id"}

// CartFieldsForRefreshSteps mirrors cartFieldsForRefreshSteps.
var CartFieldsForRefreshSteps = []string{"id", "email", "currency_code", "quantity", "subtotal", "original_item_total", "item_total", "locale", "total", "original_item_subtotal", "item_subtotal", "shipping_subtotal", "region_id", "metadata", "completed_at", "sales_channel_id", "region.*", "items.*", "items.product.id", "items.product.is_giftcard", "items.product.collection_id", "items.product.categories.id", "items.product.tags.id", "items.product.type_id", "items.product.weight", "items.variant.id", "items.variant.product.id", "items.variant.product.shipping_profile.id", "items.variant.weight", "items.variant.length", "items.variant.height", "items.variant.width", "items.variant.material", "items.adjustments.*", "items.tax_lines.*", "shipping_address.*", "shipping_methods.*", "shipping_methods.adjustments.*", "shipping_methods.tax_lines.*", "shipping_methods.shipping_option.shipping_option_type_id", "shipping_methods.shipping_option.shipping_profile_id", "customer.*", "customer.groups.*", "promotions.id", "promotions.code", "payment_collection.id", "payment_collection.raw_amount", "payment_collection.amount", "payment_collection.currency_code", "payment_collection.payment_sessions.id"}

// ProductVariantsFields mirrors productVariantsFields.
var ProductVariantsFields = []string{"id", "title", "sku", "manage_inventory", "allow_backorder", "requires_shipping", "is_discountable", "variant_option_values", "barcode", "thumbnail", "product.id", "product.title", "product.description", "product.status", "product.subtitle", "product.thumbnail", "product.type.value", "product.type.id", "product.collection.id", "product.collection.title", "product.handle", "product.discountable", "product.is_giftcard", "product.shipping_profile.id", "inventory_items.inventory_item_id", "inventory_items.required_quantity", "inventory_items.inventory.id", "inventory_items.inventory.requires_shipping", "inventory_items.inventory.location_levels.id", "inventory_items.inventory.location_levels.stocked_quantity", "inventory_items.inventory.location_levels.reserved_quantity", "inventory_items.inventory.location_levels.location_id", "inventory_items.inventory.location_levels.raw_stocked_quantity", "inventory_items.inventory.location_levels.raw_reserved_quantity", "inventory_items.inventory.location_levels.stock_locations.id", "inventory_items.inventory.location_levels.stock_locations.name", "inventory_items.inventory.location_levels.stock_locations.sales_channels.id", "inventory_items.inventory.location_levels.stock_locations.sales_channels.name"}

// Deduplicate mirrors deduplicate.
func Deduplicate(lists ...[]string) []string {
	var out []string
	seen := map[string]bool{}
	for _, l := range lists {
		for _, f := range l {
			if !seen[f] {
				seen[f] = true
				out = append(out, f)
			}
		}
	}
	return out
}
