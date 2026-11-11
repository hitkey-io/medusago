// Package modules mirrors @medusajs/medusa/src/modules: the modules and module
// providers shipped with Medusa, addressable by their package name.
package modules

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
)

// Lookup returns the exports of a configured module: its Module, or the
// package its resolve names.
func Lookup(mc config.ModuleConfig) (*modulessdk.ModuleExports, bool) {
	if mc.Module != nil {
		return mc.Module, true
	}
	exp, ok := Registry[mc.Resolve]
	return exp, ok
}

// Registry maps "@medusajs/medusa/<name>" package names to their exports.
var Registry = map[string]*modulessdk.ModuleExports{
	"@medusajs/medusa/analytics":                &Analytics,
	"@medusajs/medusa/analytics-local":          &AnalyticsLocal,
	"@medusajs/medusa/analytics-posthog":        &AnalyticsPosthog,
	"@medusajs/medusa/notification-local":       &NotificationLocal,
	"@medusajs/medusa/notification-sendgrid":    &NotificationSendgrid,
	"@medusajs/medusa/file-local":               &FileLocal,
	"@medusajs/medusa/file-s3":                  &FileS3,
	"@medusajs/medusa/file":                     &File,
	"@medusajs/medusa/auth-emailpass":           &AuthEmailpass,
	"@medusajs/medusa/auth-github":              &AuthGithub,
	"@medusajs/medusa/auth-google":              &AuthGoogle,
	"@medusajs/medusa/event-bus-local":          &EventBusLocal,
	"@medusajs/medusa/event-bus-redis":          &EventBusRedis,
	"@medusajs/medusa/fulfillment-manual":       &FulfillmentManual,
	"@medusajs/medusa/api-key":                  &ApiKey,
	"@medusajs/medusa/auth":                     &Auth,
	"@medusajs/medusa/cache-inmemory":           &CacheInmemory,
	"@medusajs/medusa/cache-redis":              &CacheRedis,
	"@medusajs/medusa/caching":                  &Caching,
	"@medusajs/medusa/caching-redis":            &CachingRedis,
	"@medusajs/medusa/cart":                     &Cart,
	"@medusajs/medusa/currency":                 &Currency,
	"@medusajs/medusa/customer":                 &Customer,
	"@medusajs/medusa/fulfillment":              &Fulfillment,
	"@medusajs/medusa/index-module":             &IndexModule,
	"@medusajs/medusa/inventory":                &Inventory,
	"@medusajs/medusa/notification":             &Notification,
	"@medusajs/medusa/order":                    &Order,
	"@medusajs/medusa/payment":                  &Payment,
	"@medusajs/medusa/payment-stripe":           &PaymentStripe,
	"@medusajs/medusa/payment-adyen":            &PaymentAdyen,
	"@medusajs/medusa/pricing":                  &Pricing,
	"@medusajs/medusa/product":                  &Product,
	"@medusajs/medusa/promotion":                &Promotion,
	"@medusajs/medusa/locking":                  &Locking,
	"@medusajs/medusa/locking-postgres":         &LockingPostgres,
	"@medusajs/medusa/locking-redis":            &LockingRedis,
	"@medusajs/medusa/search-postgres":          &SearchPostgres,
	"@medusajs/medusa/region":                   &Region,
	"@medusajs/medusa/sales-channel":            &SalesChannel,
	"@medusajs/medusa/search":                   &Search,
	"@medusajs/medusa/settings":                 &Settings,
	"@medusajs/medusa/stock-location":           &StockLocation,
	"@medusajs/medusa/store":                    &Store,
	"@medusajs/medusa/tax":                      &Tax,
	"@medusajs/medusa/translation":              &Translation,
	"@medusajs/medusa/user":                     &User,
	"@medusajs/medusa/workflow-engine-inmemory": &WorkflowEngineInmemory,
	"@medusajs/medusa/workflow-engine-redis":    &WorkflowEngineRedis,
}
