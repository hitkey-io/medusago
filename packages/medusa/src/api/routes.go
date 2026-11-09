package api

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	apikeys "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/api-keys"
	apikeyid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/api-keys/id_"
	apikeyrevoke "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/api-keys/id_/revoke"
	apikeysc "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/api-keys/id_/sales-channels"
	admincampaigns "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/campaigns"
	campaignid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/campaigns/id_"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/collections"
	collectionid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/collections/id_"
	collectionproducts "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/collections/id_/products"
	currencies "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/currencies"
	currencycode "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/currencies/code_"
	customergroups "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customer-groups"
	customergroupid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customer-groups/id_"
	customergroupcustomers "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customer-groups/id_/customers"
	admincustomers "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customers"
	customerid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customers/id_"
	customeraddresses "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customers/id_/addresses"
	customeraddress "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customers/id_/addresses/address_id_"
	customercustomergroups "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/customers/id_/customer-groups"
	featureflags "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/feature-flags"
	fulfillmentproviders "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/fulfillment-providers"
	fulfillmentprovideroptions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/fulfillment-providers/id_/options"
	fulfillmentsetid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/fulfillment-sets/id_"
	fulfillmentsetzones "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/fulfillment-sets/id_/service-zones"
	fulfillmentsetzone "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/fulfillment-sets/id_/service-zones/zone_id_"
	inventoryitems "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/inventory-items"
	inventoryitemid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/inventory-items/id_"
	inventorylevels "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/inventory-items/id_/location-levels"
	inventorylevelbatch "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/inventory-items/id_/location-levels/batch"
	inventorylevelloc "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/inventory-items/id_/location-levels/location_id_"
	inventorylevelsbatch "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/inventory-items/location-levels/batch"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/layouts/configurations"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/layouts/zone_/configuration"
	adminmodules "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/modules"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/notifications"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
	pricelists "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-lists"
	pricelistid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-lists/id_"
	pricepreferences "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-preferences"
	pricepreferenceid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-preferences/id_"
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
	adminpromotions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/promotions"
	promotionid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/promotions/id_"
	refundreasons "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/refund-reasons"
	refundreasonid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/refund-reasons/id_"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/regions"
	regionid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/regions/id_"
	adminreservations "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/reservations"
	reservationid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/reservations/id_"
	returnreasons "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/return-reasons"
	returnreasonid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/return-reasons/id_"
	saleschannels "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/sales-channels"
	saleschannelid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/sales-channels/id_"
	saleschannelproducts "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/sales-channels/id_/products"
	searchindexes "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/search-indexes"
	shippingoptiontypes "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/shipping-option-types"
	shippingoptiontypeid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/shipping-option-types/id_"
	shippingoptions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/shipping-options"
	shippingoptionid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/shipping-options/id_"
	shippingprofiles "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/shipping-profiles"
	shippingprofileid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/shipping-profiles/id_"
	stocklocations "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stock-locations"
	stocklocationid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stock-locations/id_"
	stocklocationfp "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stock-locations/id_/fulfillment-providers"
	stocklocationfs "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stock-locations/id_/fulfillment-sets"
	stocklocationsc "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stock-locations/id_/sales-channels"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stores"
	storeid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stores/id_"
	taxproviders "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/tax-providers"
	taxrates "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/tax-rates"
	taxrateid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/tax-rates/id_"
	taxraterules "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/tax-rates/id_/rules"
	taxraterule "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/tax-rates/id_/rules/rule_id_"
	taxregions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/tax-regions"
	taxregionid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/tax-regions/id_"
	adminusers "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/users"
	userid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/users/id_"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/users/me"
	workflowsexecutions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/workflows-executions"
	workflowexecutionid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/workflows-executions/id_"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/actor_type_/auth_provider_"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/actor_type_/providers"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/session"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/token/refresh"
	cloudauth "github.com/hitkey-io/medusago/packages/medusa/src/api/cloud/auth"
)

// Routes lists the route files by URL path (the file tree of src/api).
func Routes() []medusahttp.RouteFile {
	return []medusahttp.RouteFile{
		{Path: "/admin/collections", Route: collections.Route},
		{Path: "/admin/collections/:id", Route: collectionid.Route},
		{Path: "/admin/collections/:id/products", Route: collectionproducts.Route},
		{Path: "/admin/api-keys", Route: apikeys.Route},
		{Path: "/admin/api-keys/:id", Route: apikeyid.Route},
		{Path: "/admin/api-keys/:id/revoke", Route: apikeyrevoke.Route},
		{Path: "/admin/api-keys/:id/sales-channels", Route: apikeysc.Route},
		{Path: "/admin/campaigns", Route: admincampaigns.Route},
		{Path: "/admin/campaigns/:id", Route: campaignid.Route},
		{Path: "/admin/currencies", Route: currencies.Route},
		{Path: "/admin/price-lists", Route: pricelists.Route},
		{Path: "/admin/price-lists/:id", Route: pricelistid.Route},
		{Path: "/admin/promotions", Route: adminpromotions.Route},
		{Path: "/admin/promotions/:id", Route: promotionid.Route},
		{Path: "/admin/workflows-executions", Route: workflowsexecutions.Route},
		{Path: "/admin/workflows-executions/:id", Route: workflowexecutionid.Route},
		{Path: "/admin/customer-groups", Route: customergroups.Route},
		{Path: "/admin/customer-groups/:id", Route: customergroupid.Route},
		{Path: "/admin/customers", Route: admincustomers.Route},
		{Path: "/admin/customers/:id", Route: customerid.Route},
		{Path: "/admin/customers/:id/addresses", Route: customeraddresses.Route},
		{Path: "/admin/customers/:id/addresses/:address_id", Route: customeraddress.Route},
		{Path: "/admin/customers/:id/customer-groups", Route: customercustomergroups.Route},
		{Path: "/admin/customer-groups/:id/customers", Route: customergroupcustomers.Route},
		{Path: "/admin/currencies/:code", Route: currencycode.Route},
		{Path: "/admin/feature-flags", Route: featureflags.Route},
		{Path: "/admin/inventory-items", Route: inventoryitems.Route},
		{Path: "/admin/inventory-items/:id", Route: inventoryitemid.Route},
		{Path: "/admin/inventory-items/:id/location-levels", Route: inventorylevels.Route},
		{Path: "/admin/inventory-items/:id/location-levels/batch", Route: inventorylevelbatch.Route},
		{Path: "/admin/inventory-items/:id/location-levels/:location_id", Route: inventorylevelloc.Route},
		{Path: "/admin/inventory-items/location-levels/batch", Route: inventorylevelsbatch.Route},
		{Path: "/admin/reservations", Route: adminreservations.Route},
		{Path: "/admin/reservations/:id", Route: reservationid.Route},
		{Path: "/admin/stock-locations", Route: stocklocations.Route},
		{Path: "/admin/stock-locations/:id", Route: stocklocationid.Route},
		{Path: "/admin/stock-locations/:id/sales-channels", Route: stocklocationsc.Route},
		{Path: "/admin/stock-locations/:id/fulfillment-providers", Route: stocklocationfp.Route},
		{Path: "/admin/stock-locations/:id/fulfillment-sets", Route: stocklocationfs.Route},
		{Path: "/admin/shipping-profiles", Route: shippingprofiles.Route},
		{Path: "/admin/shipping-options", Route: shippingoptions.Route},
		{Path: "/admin/shipping-option-types", Route: shippingoptiontypes.Route},
		{Path: "/admin/shipping-option-types/:id", Route: shippingoptiontypeid.Route},
		{Path: "/admin/shipping-options/:id", Route: shippingoptionid.Route},
		{Path: "/admin/fulfillment-providers", Route: fulfillmentproviders.Route},
		{Path: "/admin/fulfillment-providers/:id/options", Route: fulfillmentprovideroptions.Route},
		{Path: "/admin/shipping-profiles/:id", Route: shippingprofileid.Route},
		{Path: "/admin/fulfillment-sets/:id", Route: fulfillmentsetid.Route},
		{Path: "/admin/fulfillment-sets/:id/service-zones", Route: fulfillmentsetzones.Route},
		{Path: "/admin/fulfillment-sets/:id/service-zones/:zone_id", Route: fulfillmentsetzone.Route},
		{Path: "/admin/tax-regions", Route: taxregions.Route},
		{Path: "/admin/tax-regions/:id", Route: taxregionid.Route},
		{Path: "/admin/tax-rates", Route: taxrates.Route},
		{Path: "/admin/tax-rates/:id", Route: taxrateid.Route},
		{Path: "/admin/tax-rates/:id/rules", Route: taxraterules.Route},
		{Path: "/admin/tax-rates/:id/rules/:rule_id", Route: taxraterule.Route},
		{Path: "/admin/tax-providers", Route: taxproviders.Route},
		{Path: "/admin/refund-reasons", Route: refundreasons.Route},
		{Path: "/admin/refund-reasons/:id", Route: refundreasonid.Route},
		{Path: "/admin/return-reasons", Route: returnreasons.Route},
		{Path: "/admin/return-reasons/:id", Route: returnreasonid.Route},
		{Path: "/admin/search-indexes", Route: searchindexes.Route},
		{Path: "/admin/price-preferences", Route: pricepreferences.Route},
		{Path: "/admin/price-preferences/:id", Route: pricepreferenceid.Route},
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
		{Path: "/admin/sales-channels/:id/products", Route: saleschannelproducts.Route},
		{Path: "/admin/layouts/:zone/configuration", Route: configuration.Route},
		{Path: "/admin/layouts/configurations", Route: configurations.Route},
		{Path: "/admin/modules", Route: adminmodules.Route},
		{Path: "/admin/notifications", Route: notifications.Route},
		{Path: "/admin/orders", Route: orders.Route},
		{Path: "/admin/regions", Route: regions.Route},
		{Path: "/admin/regions/:id", Route: regionid.Route},
		{Path: "/admin/sales-channels", Route: saleschannels.Route},
		{Path: "/admin/stores", Route: stores.Route},
		{Path: "/admin/stores/:id", Route: storeid.Route},
		{Path: "/admin/users", Route: adminusers.Route},
		{Path: "/admin/users/:id", Route: userid.Route},
		{Path: "/admin/users/me", Route: me.Route},
		{Path: "/auth/:actor_type/:auth_provider", Route: auth_provider_.Route},
		{Path: "/auth/:actor_type/providers", Route: providers.Route},
		{Path: "/auth/session", Route: session.Route},
		{Path: "/cloud/auth", Route: cloudauth.Route},
		{Path: "/auth/token/refresh", Route: refresh.Route},
	}
}
