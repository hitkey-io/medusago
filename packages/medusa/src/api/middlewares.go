// Package api mirrors @medusajs/medusa/src/api: the route files and the
// middlewares of defineMiddlewares, in upstream order.
package api

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/collections"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/layouts/configurations"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/notifications"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
	pricepreferences "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-preferences"
	productcategories "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-categories"
	productoptions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-options"
	producttags "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-tags"
	producttypes "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-types"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/regions"
	saleschannels "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/sales-channels"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stores"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/users"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth"
)

// Middlewares mirrors the default export of middlewares.ts.
func Middlewares() []medusahttp.MiddlewareRoute {
	var out []medusahttp.MiddlewareRoute
	// upstream order: ...auth, ..., regions, users, ..., stores, ..., sales channels
	out = append(out, auth.AuthRoutesMiddlewares...)
	out = append(out, regions.AdminRegionRoutesMiddlewares...)
	out = append(out, users.AdminUserRoutesMiddlewares...)
	out = append(out, stores.AdminStoreRoutesMiddlewares...)
	out = append(out, products.AdminProductRoutesMiddlewares...)
	out = append(out, pricepreferences.Middlewares...)
	out = append(out, collections.Middlewares...)
	out = append(out, saleschannels.AdminSalesChannelRoutesMiddlewares...)
	out = append(out, producttypes.Middlewares...)
	out = append(out, producttags.Middlewares...)
	out = append(out, productoptions.Middlewares...)
	out = append(out, notifications.AdminNotificationRoutesMiddlewares...)
	out = append(out, orders.AdminOrderRoutesMiddlewares...)
	out = append(out, productcategories.Middlewares...)
	out = append(out, configurations.LayoutConfigurationListRoutesMiddlewares...)
	return out
}
