package utils

import "github.com/hitkey-io/medusago/packages/core/utils/src/dal"

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

// CompleteCartFields mirrors completeCartFields.
var CompleteCartFields = []string{"metadata", "id", "currency_code", "email", "locale", "created_at", "updated_at", "completed_at", "total", "subtotal", "tax_total", "discount_total", "discount_tax_total", "original_total", "original_tax_total", "item_total", "item_subtotal", "item_tax_total", "sales_channel_id", "original_item_total", "original_item_subtotal", "original_item_tax_total", "shipping_total", "shipping_subtotal", "shipping_tax_total", "original_shipping_tax_total", "original_shipping_subtotal", "original_shipping_total", "raw_total", "raw_subtotal", "raw_tax_total", "raw_discount_total", "raw_discount_tax_total", "raw_original_total", "raw_original_tax_total", "raw_item_total", "raw_item_subtotal", "raw_item_tax_total", "raw_sales_channel_id", "raw_original_item_total", "raw_original_item_subtotal", "raw_original_item_tax_total", "raw_shipping_total", "raw_shipping_subtotal", "raw_shipping_tax_total", "raw_original_shipping_tax_total", "raw_original_shipping_subtotal", "raw_original_shipping_total", "items.*", "items.tax_lines.*", "items.adjustments.*", "customer.*", "shipping_methods.*", "shipping_methods.tax_lines.*", "shipping_methods.adjustments.*", "shipping_address.*", "billing_address.*", "region.*", "credit_lines.*", "payment_collection.*", "payment_collection.payment_sessions.*", "promotions.id", "items.variant.id", "items.variant.product.id", "items.variant.product.is_giftcard", "items.variant.product.shipping_profile.id", "items.variant.manage_inventory", "items.variant.allow_backorder", "items.variant.inventory_items.inventory_item_id", "items.variant.inventory_items.required_quantity", "items.variant.inventory_items.inventory.requires_shipping", "items.variant.inventory_items.inventory.location_levels.stocked_quantity", "items.variant.inventory_items.inventory.location_levels.reserved_quantity", "items.variant.inventory_items.inventory.location_levels.raw_stocked_quantity", "items.variant.inventory_items.inventory.location_levels.raw_reserved_quantity", "items.variant.inventory_items.inventory.location_levels.location_id", "items.variant.inventory_items.inventory.location_levels.stock_locations.id", "items.variant.inventory_items.inventory.location_levels.stock_locations.name", "items.variant.inventory_items.inventory.location_levels.stock_locations.sales_channels.id", "items.variant.inventory_items.inventory.location_levels.stock_locations.sales_channels.name"}

// CartFieldsForCalculateShippingOptionsPrices mirrors cartFieldsForCalculateShippingOptionsPrices.
var CartFieldsForCalculateShippingOptionsPrices = []string{"id", "locale", "currency_code", "region_id", "items.*", "items.variant.id", "items.variant.product.id", "items.variant.product.shipping_profile.id", "items.variant.weight", "items.variant.length", "items.variant.height", "items.variant.width", "items.variant.material", "items.product.id", "items.product.collection_id", "items.product.categories.id", "items.product.tags.id", "items.product.weight", "shipping_address.*"}

// FilterCartItemsByShippingProfile mirrors filterCartItemsByShippingProfile.
func FilterCartItemsByShippingProfile(items []*dal.Record, shippingProfileID any) []*dal.Record {
	out := []*dal.Record{}
	id, _ := shippingProfileID.(string)
	if len(items) == 0 || id == "" {
		return out
	}
	for _, item := range items {
		if Get(Rec(Rec(Rec(item, "variant"), "product"), "shipping_profile"), "id") == id {
			out = append(out, item)
		}
	}
	return out
}
