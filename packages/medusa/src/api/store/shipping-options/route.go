// Package shippingoptions mirrors api/store/shipping-options.
package shippingoptions

import (
	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultStoreShippingOptionsFields mirrors defaultStoreShippingOptionsFields.
var DefaultStoreShippingOptionsFields = []string{"id", "name", "price_type", "service_zone_id", "shipping_profile_id", "provider_id", "shipping_option_type_id", "metadata"}

// AllowedStoreShippingOptionExtraFields mirrors allowedStoreShippingOptionExtraFields.
var AllowedStoreShippingOptionExtraFields = []string{"data", "type", "type.id", "type.label", "type.description", "type.code", "provider", "provider.id", "provider.is_enabled", "prices", "prices.id", "prices.currency_code", "prices.amount", "prices.min_quantity", "prices.max_quantity", "prices.price_rules", "prices.price_rules.id", "prices.price_rules.attribute", "prices.price_rules.operator", "prices.price_rules.value", "calculated_price", "calculated_price.id", "calculated_price.currency_code", "calculated_price.calculated_amount", "calculated_price.calculated_amount_with_tax", "calculated_price.calculated_amount_without_tax", "calculated_price.original_amount", "calculated_price.original_amount_with_tax", "calculated_price.original_amount_without_tax", "calculated_price.is_calculated_price_price_list", "calculated_price.is_calculated_price_tax_inclusive", "calculated_price.is_original_price_price_list", "calculated_price.is_original_price_tax_inclusive", "calculated_price.calculated_price", "calculated_price.calculated_price.id", "calculated_price.calculated_price.price_list_id", "calculated_price.calculated_price.price_list_type", "calculated_price.calculated_price.min_quantity", "calculated_price.calculated_price.max_quantity", "calculated_price.original_price", "calculated_price.original_price.id", "calculated_price.original_price.price_list_id", "calculated_price.original_price.price_list_type", "calculated_price.original_price.min_quantity", "calculated_price.original_price.max_quantity"}

var listExtra = append(append([]string{}, AllowedStoreShippingOptionExtraFields...), "service_zone", "service_zone.id", "service_zone.fulfillment_set_id", "service_zone.fulfillment_set", "service_zone.fulfillment_set.id", "service_zone.fulfillment_set.type", "service_zone.fulfillment_set.location", "service_zone.fulfillment_set.location.id", "service_zone.fulfillment_set.location.address", "service_zone.fulfillment_set.location.address.id", "service_zone.fulfillment_set.location.address.company", "service_zone.fulfillment_set.location.address.address_1", "service_zone.fulfillment_set.location.address.address_2", "service_zone.fulfillment_set.location.address.city", "service_zone.fulfillment_set.location.address.country_code", "service_zone.fulfillment_set.location.address.province", "service_zone.fulfillment_set.location.address.postal_code", "service_zone.fulfillment_set.location.address.phone", "service_zone.fulfillment_set.location.address.metadata", "service_zone.fulfillment_set.location.address.created_at", "service_zone.fulfillment_set.location.address.updated_at", "service_zone.fulfillment_set.location.address.deleted_at")

// ListConfig mirrors listTransformQueryConfig (no defaults).
var ListConfig = httputils.QueryConfig{Allowed: storeutils.BuildAllowedFields(DefaultStoreShippingOptionsFields, listExtra), StoreRelationsLimit: 4, DefaultLimit: 20, IsList: true}

var fieldsSchema = zod.Object(
	zod.F("cart_id", zod.String()),
	zod.F("is_return", commonvalidators.BooleanString().Optional()),
).Strict()

// StoreGetShippingOptions mirrors StoreGetShippingOptions.
var StoreGetShippingOptions = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(20)}).
	Merge(fieldsSchema).Merge(commonvalidators.ApplyAndAndOrOperators(fieldsSchema))

// Middlewares mirrors storeShippingOptionRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/store/shipping-options", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(StoreGetShippingOptions, ListConfig)}},
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	cartID, _ := req.FilterableFields.Get("cart_id")
	isReturn, _ := req.FilterableFields.Get("is_return")
	id, _ := cartID.(string)
	b, _ := isReturn.(bool)
	opts, err := cartworkflows.ListShippingOptionsForCartWorkflow.Run(req.Ctx(), req.Scope, cartworkflows.ListShippingOptionsInput{CartID: id, IsReturn: b, Fields: req.QueryConfig.Fields})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	list := make([]any, len(opts))
	for i, o := range opts {
		list[i] = o
	}
	out.Set("shipping_options", list)
	return res.JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
