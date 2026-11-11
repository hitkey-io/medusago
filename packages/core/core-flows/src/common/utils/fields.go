// Package utils mirrors @medusajs/core-flows/src/common/utils.
package utils

// SalesChannelStockLocationCacheTags mirrors salesChannelStockLocationCacheTags:
// the cache tags for what the caching module cannot compute from a query
// resolving a sales channel's stock locations and their fulfillment sets.
// Must be passed with computeAutomaticTags.
var SalesChannelStockLocationCacheTags = []string{
	"SalesChannel:list:*",
	"LinkSalesChannelStockLocation:list:*",
	"LinkLocationFulfillmentSet:list:*",
}
