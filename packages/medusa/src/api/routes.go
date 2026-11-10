package api

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	apikeys "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/api-keys"
	apikeyid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/api-keys/id_"
	apikeyrevoke "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/api-keys/id_/revoke"
	apikeysc "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/api-keys/id_/sales-channels"
	admincampaigns "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/campaigns"
	campaignid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/campaigns/id_"
	campaignpromotions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/campaigns/id_/promotions"
	adminclaims "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/claims"
	adminclaimid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/claims/id_"
	adminclaimitems "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/claims/id_/claim-items"
	adminclaiminbounditems "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/claims/id_/inbound/items"
	adminclaiminboundshipping "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/claims/id_/inbound/shipping-method"
	adminclaimoutbounditems "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/claims/id_/outbound/items"
	adminclaimoutboundshipping "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/claims/id_/outbound/shipping-method"
	adminclaimrequest "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/claims/id_/request"
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
	adminexchanges "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/exchanges"
	adminexchangeid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/exchanges/id_"
	adminexchangeinbounditems "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/exchanges/id_/inbound/items"
	adminexchangeinboundshipping "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/exchanges/id_/inbound/shipping-method"
	adminexchangeoutbounditems "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/exchanges/id_/outbound/items"
	adminexchangeoutboundshipping "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/exchanges/id_/outbound/shipping-method"
	adminexchangerequest "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/exchanges/id_/request"
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
	admininvites "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/invites"
	inviteaccept "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/invites/accept"
	inviteid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/invites/id_"
	inviteresend "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/invites/id_/resend"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/layouts/configurations"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/layouts/zone_/configuration"
	adminmodules "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/modules"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/notifications"
	notificationid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/notifications/id_"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
	adminorderid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders/id_"
	adminorderarchive "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders/id_/archive"
	adminordercancel "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders/id_/cancel"
	adminorderchanges "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders/id_/changes"
	adminordercomplete "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders/id_/complete"
	adminordercreditlines "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders/id_/credit-lines"
	adminorderfulfillments "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders/id_/fulfillments"
	adminorderfulfillmentcancel "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders/id_/fulfillments/fulfillment_id_/cancel"
	adminorderdelivered "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders/id_/fulfillments/fulfillment_id_/mark-as-delivered"
	adminordershipments "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders/id_/fulfillments/fulfillment_id_/shipments"
	adminorderlineitems "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders/id_/line-items"
	adminorderauthorize "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders/id_/payment-sessions/authorize"
	adminorderpreview "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders/id_/preview"
	adminordershippingoptions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders/id_/shipping-options"
	adminpcs "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/payment-collections"
	adminpcid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/payment-collections/id_"
	adminpcpaid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/payment-collections/id_/mark-as-paid"
	adminpcsessions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/payment-collections/id_/payment-sessions"
	adminpayments "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/payments"
	adminpaymentid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/payments/id_"
	adminpaymentcapture "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/payments/id_/capture"
	adminpaymentrefund "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/payments/id_/refund"
	adminpaymentproviders "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/payments/payment-providers"
	adminplugins "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/plugins"
	pricelists "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-lists"
	pricelistid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-lists/id_"
	pricelistprices "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-lists/id_/prices"
	pricelistpricesbatch "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-lists/id_/prices/batch"
	pricelistproducts "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-lists/id_/products"
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
	imagevariantsbatch "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_/images/image_id_/variants/batch"
	productoptionsroute "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_/options"
	productoptionsbatch "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_/options/batch"
	productvariantsroute "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_/variants"
	productvariantsbatch "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_/variants/batch"
	variantinventorybatch "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_/variants/inventory-items/batch"
	productvariantid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_/variants/variant_id_"
	variantimagesbatch "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_/variants/variant_id_/images/batch"
	variantinventoryitems "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_/variants/variant_id_/inventory-items"
	variantinventoryitem "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products/id_/variants/variant_id_/inventory-items/inventory_item_id_"
	adminpromotions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/promotions"
	promotionid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/promotions/id_"
	promotionbuyrulesbatch "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/promotions/id_/buy-rules/batch"
	promotionruletype "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/promotions/id_/rule_type_"
	promotionrulesbatch "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/promotions/id_/rules/batch"
	promotiontargetrulesbatch "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/promotions/id_/target-rules/batch"
	promotionruleattributes "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/promotions/rule-attribute-options/rule_type_"
	promotionrulevalues "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/promotions/rule-value-options/rule_type_/rule_attribute_id_"
	refundreasons "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/refund-reasons"
	refundreasonid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/refund-reasons/id_"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/regions"
	regionid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/regions/id_"
	adminreservations "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/reservations"
	reservationid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/reservations/id_"
	returnreasons "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/return-reasons"
	returnreasonid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/return-reasons/id_"
	adminreturns "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/returns"
	adminorderedits "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/order-edits"
	adminordereditid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/order-edits/id_"
	adminordereditconfirm "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/order-edits/id_/confirm"
	adminordereditems "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/order-edits/id_/items"
	adminordereditemqty "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/order-edits/id_/items/item/item_id_"
	adminordereditrequest "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/order-edits/id_/request"
	adminordereditshipping "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/order-edits/id_/shipping-method"
	adminreturnid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/returns/id_"
	adminreturnreceive "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/returns/id_/receive"
	adminreturnreceiveitems "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/returns/id_/receive-items"
	adminreturnreceiveconfirm "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/returns/id_/receive/confirm"
	adminreturnrequest "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/returns/id_/request"
	adminreturnrequestitems "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/returns/id_/request-items"
	adminreturnshipping "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/returns/id_/shipping-method"
	saleschannels "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/sales-channels"
	saleschannelid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/sales-channels/id_"
	saleschannelproducts "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/sales-channels/id_/products"
	searchindexes "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/search-indexes"
	shippingoptiontypes "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/shipping-option-types"
	shippingoptiontypeid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/shipping-option-types/id_"
	shippingoptions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/shipping-options"
	shippingoptionid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/shipping-options/id_"
	shippingoptionrulesbatch "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/shipping-options/id_/rules/batch"
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
	adminuploads "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/uploads"
	uploadid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/uploads/id_"
	uploadpresigned "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/uploads/presigned-urls"
	adminusers "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/users"
	userid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/users/id_"
	userauthproviders "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/users/id_/auth-providers"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/users/me"
	workflowsexecutions "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/workflows-executions"
	workflowexecutionid "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/workflows-executions/id_"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/actor_type_/auth_provider_"
	authregister "github.com/hitkey-io/medusago/packages/medusa/src/api/auth/actor_type_/auth_provider_/register"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/actor_type_/providers"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/session"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/token/refresh"
	cloudauth "github.com/hitkey-io/medusago/packages/medusa/src/api/cloud/auth"
	storecarts "github.com/hitkey-io/medusago/packages/medusa/src/api/store/carts"
	storecartid "github.com/hitkey-io/medusago/packages/medusa/src/api/store/carts/id_"
	storecartcomplete "github.com/hitkey-io/medusago/packages/medusa/src/api/store/carts/id_/complete"
	storecartlineitems "github.com/hitkey-io/medusago/packages/medusa/src/api/store/carts/id_/line-items"
	storecartlineitem "github.com/hitkey-io/medusago/packages/medusa/src/api/store/carts/id_/line-items/line_id_"
	storecartpromotions "github.com/hitkey-io/medusago/packages/medusa/src/api/store/carts/id_/promotions"
	storecartshippingmethods "github.com/hitkey-io/medusago/packages/medusa/src/api/store/carts/id_/shipping-methods"
	storecollections "github.com/hitkey-io/medusago/packages/medusa/src/api/store/collections"
	storecollectionid "github.com/hitkey-io/medusago/packages/medusa/src/api/store/collections/id_"
	storecurrencies "github.com/hitkey-io/medusago/packages/medusa/src/api/store/currencies"
	storecurrencycode "github.com/hitkey-io/medusago/packages/medusa/src/api/store/currencies/code_"
	storecustomers "github.com/hitkey-io/medusago/packages/medusa/src/api/store/customers"
	storecustomerme "github.com/hitkey-io/medusago/packages/medusa/src/api/store/customers/me"
	storecustomeraddresses "github.com/hitkey-io/medusago/packages/medusa/src/api/store/customers/me/addresses"
	storecustomeraddress "github.com/hitkey-io/medusago/packages/medusa/src/api/store/customers/me/addresses/address_id_"
	storeorders "github.com/hitkey-io/medusago/packages/medusa/src/api/store/orders"
	storeorderid "github.com/hitkey-io/medusago/packages/medusa/src/api/store/orders/id_"
	storepaymentcollections "github.com/hitkey-io/medusago/packages/medusa/src/api/store/payment-collections"
	storepaymentsessions "github.com/hitkey-io/medusago/packages/medusa/src/api/store/payment-collections/id_/payment-sessions"
	storepaymentproviders "github.com/hitkey-io/medusago/packages/medusa/src/api/store/payment-providers"
	storecategories "github.com/hitkey-io/medusago/packages/medusa/src/api/store/product-categories"
	storecategoryid "github.com/hitkey-io/medusago/packages/medusa/src/api/store/product-categories/id_"
	storeoptions "github.com/hitkey-io/medusago/packages/medusa/src/api/store/product-options"
	storeoptionid "github.com/hitkey-io/medusago/packages/medusa/src/api/store/product-options/id_"
	storetags "github.com/hitkey-io/medusago/packages/medusa/src/api/store/product-tags"
	storetagid "github.com/hitkey-io/medusago/packages/medusa/src/api/store/product-tags/id_"
	storetypes "github.com/hitkey-io/medusago/packages/medusa/src/api/store/product-types"
	storetypeid "github.com/hitkey-io/medusago/packages/medusa/src/api/store/product-types/id_"
	storevariants "github.com/hitkey-io/medusago/packages/medusa/src/api/store/product-variants"
	storevariantid "github.com/hitkey-io/medusago/packages/medusa/src/api/store/product-variants/id_"
	storeproducts "github.com/hitkey-io/medusago/packages/medusa/src/api/store/products"
	storeproductid "github.com/hitkey-io/medusago/packages/medusa/src/api/store/products/id_"
	storeregions "github.com/hitkey-io/medusago/packages/medusa/src/api/store/regions"
	storeregionid "github.com/hitkey-io/medusago/packages/medusa/src/api/store/regions/id_"
	storereasons "github.com/hitkey-io/medusago/packages/medusa/src/api/store/return-reasons"
	storereasonid "github.com/hitkey-io/medusago/packages/medusa/src/api/store/return-reasons/id_"
	storeshippingoptions "github.com/hitkey-io/medusago/packages/medusa/src/api/store/shipping-options"
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
		{Path: "/admin/campaigns/:id/promotions", Route: campaignpromotions.Route},
		{Path: "/admin/currencies", Route: currencies.Route},
		{Path: "/admin/price-lists", Route: pricelists.Route},
		{Path: "/admin/price-lists/:id", Route: pricelistid.Route},
		{Path: "/admin/price-lists/:id/prices", Route: pricelistprices.Route},
		{Path: "/admin/price-lists/:id/prices/batch", Route: pricelistpricesbatch.Route},
		{Path: "/admin/price-lists/:id/products", Route: pricelistproducts.Route},
		{Path: "/admin/promotions", Route: adminpromotions.Route},
		{Path: "/admin/promotions/:id", Route: promotionid.Route},
		{Path: "/admin/promotions/:id/:rule_type", Route: promotionruletype.Route},
		{Path: "/admin/promotions/rule-attribute-options/:rule_type", Route: promotionruleattributes.Route},
		{Path: "/admin/promotions/rule-value-options/:rule_type/:rule_attribute_id", Route: promotionrulevalues.Route},
		{Path: "/admin/promotions/:id/rules/batch", Route: promotionrulesbatch.Route},
		{Path: "/admin/promotions/:id/target-rules/batch", Route: promotiontargetrulesbatch.Route},
		{Path: "/admin/promotions/:id/buy-rules/batch", Route: promotionbuyrulesbatch.Route},
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
		{Path: "/admin/shipping-options/:id/rules/batch", Route: shippingoptionrulesbatch.Route},
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
		{Path: "/admin/products/:id/variants/:variant_id/inventory-items", Route: variantinventoryitems.Route},
		{Path: "/admin/products/:id/variants/:variant_id/inventory-items/:inventory_item_id", Route: variantinventoryitem.Route},
		{Path: "/admin/products/:id/variants/inventory-items/batch", Route: variantinventorybatch.Route},
		{Path: "/admin/products/:id/images/:image_id/variants/batch", Route: imagevariantsbatch.Route},
		{Path: "/admin/products/:id/variants/:variant_id/images/batch", Route: variantimagesbatch.Route},
		{Path: "/admin/sales-channels/:id", Route: saleschannelid.Route},
		{Path: "/admin/sales-channels/:id/products", Route: saleschannelproducts.Route},
		{Path: "/admin/layouts/:zone/configuration", Route: configuration.Route},
		{Path: "/admin/layouts/configurations", Route: configurations.Route},
		{Path: "/admin/modules", Route: adminmodules.Route},
		{Path: "/admin/notifications", Route: notifications.Route},
		{Path: "/admin/orders", Route: orders.Route},
		{Path: "/admin/orders/:id", Route: adminorderid.Route},
		{Path: "/admin/orders/:id/cancel", Route: adminordercancel.Route},
		{Path: "/admin/orders/:id/credit-lines", Route: adminordercreditlines.Route},
		{Path: "/admin/orders/:id/shipping-options", Route: adminordershippingoptions.Route},
		{Path: "/admin/orders/:id/payment-sessions/authorize", Route: adminorderauthorize.Route},
		{Path: "/admin/orders/:id/complete", Route: adminordercomplete.Route},
		{Path: "/admin/orders/:id/archive", Route: adminorderarchive.Route},
		{Path: "/admin/payments", Route: adminpayments.Route},
		{Path: "/admin/payment-collections", Route: adminpcs.Route},
		{Path: "/admin/payment-collections/:id", Route: adminpcid.Route},
		{Path: "/admin/payment-collections/:id/mark-as-paid", Route: adminpcpaid.Route},
		{Path: "/admin/payment-collections/:id/payment-sessions", Route: adminpcsessions.Route},
		{Path: "/admin/payments/payment-providers", Route: adminpaymentproviders.Route},
		{Path: "/admin/payments/:id", Route: adminpaymentid.Route},
		{Path: "/admin/payments/:id/capture", Route: adminpaymentcapture.Route},
		{Path: "/admin/payments/:id/refund", Route: adminpaymentrefund.Route},
		{Path: "/admin/orders/:id/fulfillments", Route: adminorderfulfillments.Route},
		{Path: "/admin/orders/:id/fulfillments/:fulfillment_id/cancel", Route: adminorderfulfillmentcancel.Route},
		{Path: "/admin/orders/:id/fulfillments/:fulfillment_id/shipments", Route: adminordershipments.Route},
		{Path: "/admin/orders/:id/fulfillments/:fulfillment_id/mark-as-delivered", Route: adminorderdelivered.Route},
		{Path: "/admin/orders/:id/changes", Route: adminorderchanges.Route},
		{Path: "/admin/orders/:id/preview", Route: adminorderpreview.Route},
		{Path: "/admin/orders/:id/line-items", Route: adminorderlineitems.Route},
		{Path: "/admin/returns", Route: adminreturns.Route},
		{Path: "/admin/claims", Route: adminclaims.Route},
		{Path: "/admin/exchanges", Route: adminexchanges.Route},
		{Path: "/admin/order-edits", Route: adminorderedits.Route},
		{Path: "/admin/order-edits/:id", Route: adminordereditid.Route},
		{Path: "/admin/order-edits/:id/items", Route: adminordereditems.Route},
		{Path: "/admin/order-edits/:id/items/item/:item_id", Route: adminordereditemqty.Route},
		{Path: "/admin/order-edits/:id/shipping-method", Route: adminordereditshipping.Route},
		{Path: "/admin/order-edits/:id/request", Route: adminordereditrequest.Route},
		{Path: "/admin/order-edits/:id/confirm", Route: adminordereditconfirm.Route},
		{Path: "/admin/exchanges/:id", Route: adminexchangeid.Route},
		{Path: "/admin/exchanges/:id/inbound/items", Route: adminexchangeinbounditems.Route},
		{Path: "/admin/exchanges/:id/inbound/shipping-method", Route: adminexchangeinboundshipping.Route},
		{Path: "/admin/exchanges/:id/outbound/items", Route: adminexchangeoutbounditems.Route},
		{Path: "/admin/exchanges/:id/outbound/shipping-method", Route: adminexchangeoutboundshipping.Route},
		{Path: "/admin/exchanges/:id/request", Route: adminexchangerequest.Route},
		{Path: "/admin/claims/:id", Route: adminclaimid.Route},
		{Path: "/admin/claims/:id/claim-items", Route: adminclaimitems.Route},
		{Path: "/admin/claims/:id/inbound/items", Route: adminclaiminbounditems.Route},
		{Path: "/admin/claims/:id/inbound/shipping-method", Route: adminclaiminboundshipping.Route},
		{Path: "/admin/claims/:id/outbound/items", Route: adminclaimoutbounditems.Route},
		{Path: "/admin/claims/:id/outbound/shipping-method", Route: adminclaimoutboundshipping.Route},
		{Path: "/admin/claims/:id/request", Route: adminclaimrequest.Route},
		{Path: "/admin/returns/:id", Route: adminreturnid.Route},
		{Path: "/admin/returns/:id/request-items", Route: adminreturnrequestitems.Route},
		{Path: "/admin/returns/:id/shipping-method", Route: adminreturnshipping.Route},
		{Path: "/admin/returns/:id/request", Route: adminreturnrequest.Route},
		{Path: "/admin/returns/:id/receive", Route: adminreturnreceive.Route},
		{Path: "/admin/returns/:id/receive/confirm", Route: adminreturnreceiveconfirm.Route},
		{Path: "/admin/returns/:id/receive-items", Route: adminreturnreceiveitems.Route},
		{Path: "/admin/regions", Route: regions.Route},
		{Path: "/admin/regions/:id", Route: regionid.Route},
		{Path: "/admin/sales-channels", Route: saleschannels.Route},
		{Path: "/admin/stores", Route: stores.Route},
		{Path: "/admin/stores/:id", Route: storeid.Route},
		{Path: "/admin/users", Route: adminusers.Route},
		{Path: "/admin/users/:id", Route: userid.Route},
		{Path: "/admin/users/me", Route: me.Route},
		{Path: "/admin/users/:id/auth-providers", Route: userauthproviders.Route},
		{Path: "/admin/invites", Route: admininvites.Route},
		{Path: "/admin/uploads", Route: adminuploads.Route},
		{Path: "/store/carts", Route: storecarts.Route},
		{Path: "/store/carts/:id", Route: storecartid.Route},
		{Path: "/store/carts/:id/line-items", Route: storecartlineitems.Route},
		{Path: "/store/carts/:id/line-items/:line_id", Route: storecartlineitem.Route},
		{Path: "/store/carts/:id/promotions", Route: storecartpromotions.Route},
		{Path: "/store/carts/:id/shipping-methods", Route: storecartshippingmethods.Route},
		{Path: "/store/carts/:id/complete", Route: storecartcomplete.Route},
		{Path: "/store/orders", Route: storeorders.Route},
		{Path: "/store/orders/:id", Route: storeorderid.Route},
		{Path: "/store/customers", Route: storecustomers.Route},
		{Path: "/store/customers/me", Route: storecustomerme.Route},
		{Path: "/store/customers/me/addresses", Route: storecustomeraddresses.Route},
		{Path: "/store/customers/me/addresses/:address_id", Route: storecustomeraddress.Route},
		{Path: "/store/payment-collections", Route: storepaymentcollections.Route},
		{Path: "/store/payment-collections/:id/payment-sessions", Route: storepaymentsessions.Route},
		{Path: "/store/payment-providers", Route: storepaymentproviders.Route},
		{Path: "/store/product-variants", Route: storevariants.Route},
		{Path: "/store/product-variants/:id", Route: storevariantid.Route},
		{Path: "/store/products", Route: storeproducts.Route},
		{Path: "/store/shipping-options", Route: storeshippingoptions.Route},
		{Path: "/store/products/:id", Route: storeproductid.Route},
		{Path: "/store/regions", Route: storeregions.Route},
		{Path: "/store/regions/:id", Route: storeregionid.Route},
		{Path: "/store/currencies", Route: storecurrencies.Route},
		{Path: "/store/currencies/:code", Route: storecurrencycode.Route},
		{Path: "/store/collections", Route: storecollections.Route},
		{Path: "/store/collections/:id", Route: storecollectionid.Route},
		{Path: "/store/product-categories", Route: storecategories.Route},
		{Path: "/store/product-categories/:id", Route: storecategoryid.Route},
		{Path: "/store/product-tags", Route: storetags.Route},
		{Path: "/store/product-tags/:id", Route: storetagid.Route},
		{Path: "/store/product-types", Route: storetypes.Route},
		{Path: "/store/product-types/:id", Route: storetypeid.Route},
		{Path: "/store/return-reasons", Route: storereasons.Route},
		{Path: "/store/return-reasons/:id", Route: storereasonid.Route},
		{Path: "/store/product-options", Route: storeoptions.Route},
		{Path: "/store/product-options/:id", Route: storeoptionid.Route},
		{Path: "/admin/notifications/:id", Route: notificationid.Route},
		{Path: "/admin/plugins", Route: adminplugins.Route},
		{Path: "/admin/uploads/:id", Route: uploadid.Route},
		{Path: "/admin/uploads/presigned-urls", Route: uploadpresigned.Route},
		{Path: "/admin/invites/accept", Route: inviteaccept.Route},
		{Path: "/admin/invites/:id", Route: inviteid.Route},
		{Path: "/admin/invites/:id/resend", Route: inviteresend.Route},
		{Path: "/auth/:actor_type/:auth_provider", Route: auth_provider_.Route},
		{Path: "/auth/:actor_type/:auth_provider/register", Route: authregister.Route},
		{Path: "/auth/:actor_type/providers", Route: providers.Route},
		{Path: "/auth/session", Route: session.Route},
		{Path: "/cloud/auth", Route: cloudauth.Route},
		{Path: "/auth/token/refresh", Route: refresh.Route},
	}
}
