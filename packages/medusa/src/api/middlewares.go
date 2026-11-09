// Package api mirrors @medusajs/medusa/src/api: the route files and the
// middlewares of defineMiddlewares, in upstream order.
package api

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	apikeys "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/api-keys"
	admincampaigns "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/campaigns"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/collections"
	currencies "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/currencies"
	customergroups "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customer-groups"
	admincustomers "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customers"
	inventoryitems "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/inventory-items"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/layouts/configurations"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/notifications"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
	pricelists "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-lists"
	pricepreferences "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-preferences"
	productcategories "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-categories"
	productoptions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-options"
	producttags "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-tags"
	producttypes "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-types"
	productvariants "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-variants"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products"
	adminpromotions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/promotions"
	refundreasons "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/refund-reasons"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/regions"
	adminreservations "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/reservations"
	returnreasons "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/return-reasons"
	saleschannels "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/sales-channels"
	stocklocations "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stock-locations"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stores"
	taxproviders "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/tax-providers"
	taxrates "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/tax-rates"
	taxregions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/tax-regions"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/users"
	workflowsexecutions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/workflows-executions"
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
	out = append(out, productvariants.Middlewares...)
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
	out = append(out, currencies.Middlewares...)
	out = append(out, pricelists.Middlewares...)
	out = append(out, adminpromotions.Middlewares...)
	out = append(out, admincampaigns.Middlewares...)
	out = append(out, workflowsexecutions.Middlewares...)
	out = append(out, stocklocations.Middlewares...)
	out = append(out, inventoryitems.Middlewares...)
	out = append(out, adminreservations.Middlewares...)
	out = append(out, taxrates.Middlewares...)
	out = append(out, taxregions.Middlewares...)
	out = append(out, taxproviders.Middlewares...)
	out = append(out, apikeys.Middlewares...)
	out = append(out, admincustomers.Middlewares...)
	out = append(out, customergroups.Middlewares...)
	out = append(out, returnreasons.Middlewares...)
	out = append(out, refundreasons.Middlewares...)
	return out
}
