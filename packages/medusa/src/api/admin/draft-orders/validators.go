package draftorders

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

func metadata() *zod.Schema { return zod.Record(zod.String(), zod.Unknown()) }

// AdminGetDraftOrderParams mirrors AdminGetDraftOrderParams.
var AdminGetDraftOrderParams = utils.CreateSelectParams()

var adminGetDraftOrdersParamsFields = zod.Object(
	zod.F("id", zod.Union(zod.String(), zod.Array(zod.String())).Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("q", zod.String().Optional()),
	zod.F("region_id", zod.Union(zod.String(), zod.Array(zod.String())).Optional()),
	zod.F("sales_channel_id", zod.Array(zod.String()).Optional()),
	zod.F("customer_id", zod.Union(zod.String(), zod.Array(zod.String())).Optional()),
)

// AdminGetDraftOrdersParams mirrors AdminGetDraftOrdersParams.
var AdminGetDraftOrdersParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(50), Offset: utils.Num(0)}).
	Merge(adminGetDraftOrdersParamsFields).
	Merge(commonvalidators.ApplyAndAndOrOperators(adminGetDraftOrdersParamsFields))

var shippingMethod = zod.Object(
	zod.F("shipping_method_id", zod.String().Nullish()),
	zod.F("name", zod.String()),
	zod.F("shipping_option_id", zod.String()),
	zod.F("data", metadata().Optional()),
	zod.F("amount", commonvalidators.BigNumberInput()),
)

var item = zod.Object(
	zod.F("title", zod.String().Nullish()),
	zod.F("variant_sku", zod.String().Nullish()),
	zod.F("variant_barcode", zod.String().Nullish()),
	zod.F("sku", zod.String().Nullish()),
	zod.F("barcode", zod.String().Nullish()),
	zod.F("variant_id", zod.String().Nullish()),
	zod.F("unit_price", commonvalidators.BigNumberInput().Nullish()),
	zod.F("quantity", zod.Number()),
	zod.F("metadata", metadata().Nullish()),
)

var createDraftOrder = zod.Object(
	zod.F("status", zod.Enum("completed").Optional()),
	zod.F("sales_channel_id", zod.String().Nullish()),
	zod.F("email", zod.String().Nullish()),
	zod.F("customer_id", zod.String().Nullish()),
	zod.F("billing_address", zod.Union(commonvalidators.AddressPayload(), zod.String()).Optional()),
	zod.F("shipping_address", zod.Union(commonvalidators.AddressPayload(), zod.String()).Optional()),
	zod.F("items", zod.Array(item).Optional()),
	zod.F("region_id", zod.String()),
	zod.F("promo_codes", zod.Array(zod.String()).Optional()),
	zod.F("currency_code", zod.String().Nullish()),
	zod.F("no_notification_order", zod.Boolean().Optional()),
	zod.F("shipping_methods", zod.Array(shippingMethod).Optional()),
	zod.F("locale", zod.String().Optional()),
	zod.F("metadata", metadata().Nullish()),
).Strict()

// AdminCreateDraftOrder mirrors AdminCreateDraftOrder.
var AdminCreateDraftOrder = utils.WithAdditionalData(createDraftOrder)

// AdminUpdateDraftOrder mirrors AdminUpdateDraftOrder.
var AdminUpdateDraftOrder = zod.Object(
	zod.F("email", zod.String().Optional()),
	zod.F("customer_id", zod.String().Optional()),
	zod.F("sales_channel_id", zod.String().Optional()),
	zod.F("shipping_address", commonvalidators.AddressPayload().Optional()),
	zod.F("billing_address", commonvalidators.AddressPayload().Optional()),
	zod.F("metadata", metadata().Nullish()),
	zod.F("locale", zod.String().Optional()),
)
