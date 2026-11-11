// Package modulessdk mirrors @medusajs/utils/src/modules-sdk.
package modulessdk

// Module keys, mirroring Modules from definition.ts.
const (
	ANALYTICS       = "analytics"
	AUTH            = "auth"
	CACHE           = "cache"
	CART            = "cart"
	CUSTOMER        = "customer"
	EVENT_BUS       = "event_bus"
	INVENTORY       = "inventory"
	LINK            = "link_modules"
	PAYMENT         = "payment"
	PRICING         = "pricing"
	PRODUCT         = "product"
	PROMOTION       = "promotion"
	SALES_CHANNEL   = "sales_channel"
	TAX             = "tax"
	FULFILLMENT     = "fulfillment"
	STOCK_LOCATION  = "stock_location"
	USER            = "user"
	WORKFLOW_ENGINE = "workflows"
	REGION          = "region"
	ORDER           = "order"
	API_KEY         = "api_key"
	STORE           = "store"
	CURRENCY        = "currency"
	FILE            = "file"
	NOTIFICATION    = "notification"
	INDEX           = "index"
	SEARCH          = "search"
	LOCKING         = "locking"
	SETTINGS        = "settings"
	CACHING         = "caching"
	TRANSLATION     = "translation"
	RBAC            = "rbac"
)

// MODULE_PACKAGE_NAMES maps module keys to their default package, mirroring
// MODULE_PACKAGE_NAMES from definition.ts.
var MODULE_PACKAGE_NAMES = map[string]string{
	ANALYTICS:       "@medusajs/medusa/analytics",
	AUTH:            "@medusajs/medusa/auth",
	CACHE:           "@medusajs/medusa/cache-inmemory",
	CART:            "@medusajs/medusa/cart",
	CUSTOMER:        "@medusajs/medusa/customer",
	EVENT_BUS:       "@medusajs/medusa/event-bus-local",
	INVENTORY:       "@medusajs/medusa/inventory",
	LINK:            "@medusajs/medusa/link-modules",
	PAYMENT:         "@medusajs/medusa/payment",
	PRICING:         "@medusajs/medusa/pricing",
	PRODUCT:         "@medusajs/medusa/product",
	PROMOTION:       "@medusajs/medusa/promotion",
	SALES_CHANNEL:   "@medusajs/medusa/sales-channel",
	FULFILLMENT:     "@medusajs/medusa/fulfillment",
	STOCK_LOCATION:  "@medusajs/medusa/stock-location",
	TAX:             "@medusajs/medusa/tax",
	USER:            "@medusajs/medusa/user",
	WORKFLOW_ENGINE: "@medusajs/medusa/workflow-engine-inmemory",
	REGION:          "@medusajs/medusa/region",
	ORDER:           "@medusajs/medusa/order",
	API_KEY:         "@medusajs/medusa/api-key",
	STORE:           "@medusajs/medusa/store",
	CURRENCY:        "@medusajs/medusa/currency",
	FILE:            "@medusajs/medusa/file",
	NOTIFICATION:    "@medusajs/medusa/notification",
	INDEX:           "@medusajs/medusa/index-module",
	SEARCH:          "@medusajs/medusa/search",
	LOCKING:         "@medusajs/medusa/locking",
	SETTINGS:        "@medusajs/medusa/settings",
	CACHING:         "@medusajs/medusa/caching",
	TRANSLATION:     "@medusajs/medusa/translation",
	RBAC:            "@medusajs/medusa/rbac",
}

// TEMPORARY_REDIS_MODULE_PACKAGE_NAMES mirrors
// TEMPORARY_REDIS_MODULE_PACKAGE_NAMES.
var TEMPORARY_REDIS_MODULE_PACKAGE_NAMES = map[string]string{
	EVENT_BUS:       "@medusajs/medusa/event-bus-redis",
	CACHE:           "@medusajs/medusa/cache-redis",
	WORKFLOW_ENGINE: "@medusajs/medusa/workflow-engine-redis",
	LOCKING:         "@medusajs/medusa/locking-redis",
}

// ReversedModulePackageName mirrors REVERSED_MODULE_PACKAGE_NAMES[resolve]
// (including the temporary Redis package names): the module key of a
// package, "" when unknown.
func ReversedModulePackageName(resolve string) string {
	for k, v := range MODULE_PACKAGE_NAMES {
		if v == resolve {
			return k
		}
	}
	for k, v := range TEMPORARY_REDIS_MODULE_PACKAGE_NAMES {
		if v == resolve {
			return k
		}
	}
	return ""
}
