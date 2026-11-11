package utils

// ProductVariantsCacheTags mirrors productVariantsCacheTags (order/utils/fields.ts):
// the cache tags for what the caching module cannot compute from a response
// shaped by productVariantsFields. Must be passed with computeAutomaticTags.
var ProductVariantsCacheTags = []string{
	// Reservations adjust inventory levels without emitting inventory level events.
	"ReservationItem:list:*",
	"LinkProductVariantInventoryItem:list:*",
	"LinkSalesChannelStockLocation:list:*",
}
