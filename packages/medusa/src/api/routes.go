package api

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	apikeys "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/api-keys"
	apikeyid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/api-keys/id_"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/collections"
	collectionid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/collections/id_"
	collectionproducts "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/collections/id_/products"
	currencies "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/currencies"
	currencycode "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/currencies/code_"
	customergroups "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customer-groups"
	customergroupid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customer-groups/id_"
	admincustomers "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customers"
	customerid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customers/id_"
	featureflags "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/feature-flags"
	inventoryitems "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/inventory-items"
	inventoryitemid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/inventory-items/id_"
	inventorylevels "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/inventory-items/id_/location-levels"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/layouts/configurations"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/layouts/zone_/configuration"
	adminmodules "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/modules"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/notifications"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
	pricepreferences "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-preferences"
	productcategories "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-categories"
	categoryid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-categories/id_"
	categoryproducts "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-categories/id_/products"
	productoptions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-options"
	productoptionid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-options/id_"
	productoptionvalues "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-options/id_/values"
	productoptionvalueid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-options/id_/values/value_id_"
	producttags "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-tags"
	producttagid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-tags/id_"
	producttypes "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-types"
	producttypeid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-types/id_"
	productvariantslist "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-variants"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products"
	productid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_"
	productoptionsroute "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_/options"
	productoptionsbatch "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_/options/batch"
	productvariantsroute "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_/variants"
	productvariantsbatch "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_/variants/batch"
	productvariantid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_/variants/variant_id_"
	refundreasons "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/refund-reasons"
	refundreasonid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/refund-reasons/id_"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/regions"
	adminreservations "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/reservations"
	reservationid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/reservations/id_"
	returnreasons "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/return-reasons"
	returnreasonid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/return-reasons/id_"
	saleschannels "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/sales-channels"
	saleschannelid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/sales-channels/id_"
	searchindexes "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/search-indexes"
	stocklocations "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stock-locations"
	stocklocationid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stock-locations/id_"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stores"
	taxregions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/tax-regions"
	taxregionid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/tax-regions/id_"
	adminusers "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/users"
	userid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/users/id_"
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
		{Path: "/admin/collections/:id/products", Route: collectionproducts.Route},
		{Path: "/admin/api-keys", Route: apikeys.Route},
		{Path: "/admin/api-keys/:id", Route: apikeyid.Route},
		{Path: "/admin/currencies", Route: currencies.Route},
		{Path: "/admin/customer-groups", Route: customergroups.Route},
		{Path: "/admin/customer-groups/:id", Route: customergroupid.Route},
		{Path: "/admin/customers", Route: admincustomers.Route},
		{Path: "/admin/customers/:id", Route: customerid.Route},
		{Path: "/admin/currencies/:code", Route: currencycode.Route},
		{Path: "/admin/feature-flags", Route: featureflags.Route},
		{Path: "/admin/inventory-items", Route: inventoryitems.Route},
		{Path: "/admin/inventory-items/:id", Route: inventoryitemid.Route},
		{Path: "/admin/inventory-items/:id/location-levels", Route: inventorylevels.Route},
		{Path: "/admin/reservations", Route: adminreservations.Route},
		{Path: "/admin/reservations/:id", Route: reservationid.Route},
		{Path: "/admin/stock-locations", Route: stocklocations.Route},
		{Path: "/admin/stock-locations/:id", Route: stocklocationid.Route},
		{Path: "/admin/tax-regions", Route: taxregions.Route},
		{Path: "/admin/tax-regions/:id", Route: taxregionid.Route},
		{Path: "/admin/refund-reasons", Route: refundreasons.Route},
		{Path: "/admin/refund-reasons/:id", Route: refundreasonid.Route},
		{Path: "/admin/return-reasons", Route: returnreasons.Route},
		{Path: "/admin/return-reasons/:id", Route: returnreasonid.Route},
		{Path: "/admin/search-indexes", Route: searchindexes.Route},
		{Path: "/admin/price-preferences", Route: pricepreferences.Route},
		{Path: "/admin/product-categories", Route: productcategories.Route},
		{Path: "/admin/product-categories/:id", Route: categoryid.Route},
		{Path: "/admin/product-categories/:id/products", Route: categoryproducts.Route},
		{Path: "/admin/product-options", Route: productoptions.Route},
		{Path: "/admin/product-options/:id", Route: productoptionid.Route},
		{Path: "/admin/product-options/:id/values", Route: productoptionvalues.Route},
		{Path: "/admin/product-options/:id/values/:value_id", Route: productoptionvalueid.Route},
		{Path: "/admin/product-tags", Route: producttags.Route},
		{Path: "/admin/product-tags/:id", Route: producttagid.Route},
		{Path: "/admin/product-types", Route: producttypes.Route},
		{Path: "/admin/product-types/:id", Route: producttypeid.Route},
		{Path: "/admin/product-variants", Route: productvariantslist.Route},
		{Path: "/admin/products", Route: products.Route},
		{Path: "/admin/products/:id", Route: productid.Route},
		{Path: "/admin/products/:id/options", Route: productoptionsroute.Route},
		{Path: "/admin/products/:id/options/batch", Route: productoptionsbatch.Route},
		{Path: "/admin/products/:id/variants", Route: productvariantsroute.Route},
		{Path: "/admin/products/:id/variants/batch", Route: productvariantsbatch.Route},
		{Path: "/admin/products/:id/variants/:variant_id", Route: productvariantid.Route},
		{Path: "/admin/sales-channels/:id", Route: saleschannelid.Route},
		{Path: "/admin/layouts/:zone/configuration", Route: configuration.Route},
		{Path: "/admin/layouts/configurations", Route: configurations.Route},
		{Path: "/admin/modules", Route: adminmodules.Route},
		{Path: "/admin/notifications", Route: notifications.Route},
		{Path: "/admin/orders", Route: orders.Route},
		{Path: "/admin/regions", Route: regions.Route},
		{Path: "/admin/sales-channels", Route: saleschannels.Route},
		{Path: "/admin/stores", Route: stores.Route},
		{Path: "/admin/users", Route: adminusers.Route},
		{Path: "/admin/users/:id", Route: userid.Route},
		{Path: "/admin/users/me", Route: me.Route},
		{Path: "/auth/:actor_type/:auth_provider", Route: auth_provider_.Route},
		{Path: "/auth/:actor_type/providers", Route: providers.Route},
		{Path: "/auth/session", Route: session.Route},
		{Path: "/auth/token/refresh", Route: refresh.Route},
	}
}
