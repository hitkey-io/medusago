package api

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/collections"
	collectionid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/collections/id_"
	featureflags "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/feature-flags"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/layouts/configurations"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/layouts/zone_/configuration"
	adminmodules "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/modules"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/notifications"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
	pricepreferences "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-preferences"
	productcategories "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-categories"
	productoptions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-options"
	producttags "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-tags"
	producttagid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-tags/id_"
	producttypes "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-types"
	producttypeid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-types/id_"
	productvariantslist "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-variants"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products"
	productid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_"
	productvariantsroute "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_/variants"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/regions"
	saleschannels "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/sales-channels"
	saleschannelid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/sales-channels/id_"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stores"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/users/me"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/actor_type_/auth_provider_"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/actor_type_/providers"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/session"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/token/refresh"
)

// Routes lists the route files by URL path (the file tree of src/api).
func Routes() []medusahttp.RouteFile {
	return []medusahttp.RouteFile{
		{Path: "/admin/collections", Route: collections.Route},
		{Path: "/admin/collections/:id", Route: collectionid.Route},
		{Path: "/admin/feature-flags", Route: featureflags.Route},
		{Path: "/admin/price-preferences", Route: pricepreferences.Route},
		{Path: "/admin/product-categories", Route: productcategories.Route},
		{Path: "/admin/product-options", Route: productoptions.Route},
		{Path: "/admin/product-tags", Route: producttags.Route},
		{Path: "/admin/product-tags/:id", Route: producttagid.Route},
		{Path: "/admin/product-types", Route: producttypes.Route},
		{Path: "/admin/product-types/:id", Route: producttypeid.Route},
		{Path: "/admin/product-variants", Route: productvariantslist.Route},
		{Path: "/admin/products", Route: products.Route},
		{Path: "/admin/products/:id", Route: productid.Route},
		{Path: "/admin/products/:id/variants", Route: productvariantsroute.Route},
		{Path: "/admin/sales-channels/:id", Route: saleschannelid.Route},
		{Path: "/admin/layouts/:zone/configuration", Route: configuration.Route},
		{Path: "/admin/layouts/configurations", Route: configurations.Route},
		{Path: "/admin/modules", Route: adminmodules.Route},
		{Path: "/admin/notifications", Route: notifications.Route},
		{Path: "/admin/orders", Route: orders.Route},
		{Path: "/admin/regions", Route: regions.Route},
		{Path: "/admin/sales-channels", Route: saleschannels.Route},
		{Path: "/admin/stores", Route: stores.Route},
		{Path: "/admin/users/me", Route: me.Route},
		{Path: "/auth/:actor_type/:auth_provider", Route: auth_provider_.Route},
		{Path: "/auth/:actor_type/providers", Route: providers.Route},
		{Path: "/auth/session", Route: session.Route},
		{Path: "/auth/token/refresh", Route: refresh.Route},
	}
}
