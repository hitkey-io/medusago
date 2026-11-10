package exchanges

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/returns"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

func metadata() *zod.Schema { return zod.Record(zod.String(), zod.Unknown()) }

// AdminGetOrdersOrderParams mirrors AdminGetOrdersOrderParams (exchanges).
var AdminGetOrdersOrderParams = returns.AdminGetOrdersOrderParams

// AdminGetOrdersParams mirrors AdminGetOrdersParams (exchanges).
var AdminGetOrdersParams = returns.AdminGetOrdersParams

// AdminPostOrderExchangesReqSchema mirrors AdminPostOrderExchangesReqSchema.
var AdminPostOrderExchangesReqSchema = zod.Object(
	zod.F("order_id", zod.String()),
	zod.F("description", zod.String().Optional()),
	zod.F("internal_note", zod.String().Optional()),
	zod.F("metadata", metadata().Nullish()),
)

// AdminPostExchangesShippingReqSchema mirrors AdminPostExchangesShippingReqSchema.
var AdminPostExchangesShippingReqSchema = zod.Object(
	zod.F("shipping_option_id", zod.String()),
	zod.F("custom_amount", zod.Number().Optional()),
	zod.F("description", zod.String().Optional()),
	zod.F("internal_note", zod.String().Optional()),
	zod.F("metadata", metadata().Optional()),
)

// AdminPostExchangesAddItemsReqSchema mirrors AdminPostExchangesAddItemsReqSchema.
var AdminPostExchangesAddItemsReqSchema = zod.Object(
	zod.F("items", zod.Array(zod.Object(
		zod.F("variant_id", zod.String()),
		zod.F("quantity", zod.Number()),
		zod.F("unit_price", zod.Number().Optional()),
		zod.F("internal_note", zod.String().Optional()),
		zod.F("allow_backorder", zod.Boolean().Optional()),
		zod.F("metadata", metadata().Optional()),
	))),
)

// AdminPostExchangesReturnRequestItemsReqSchema mirrors AdminPostExchangesReturnRequestItemsReqSchema.
var AdminPostExchangesReturnRequestItemsReqSchema = zod.Object(
	zod.F("location_id", zod.String().Optional()),
	zod.F("items", zod.Array(zod.Object(
		zod.F("id", zod.String()),
		zod.F("quantity", zod.Number()),
		zod.F("description", zod.String().Optional()),
		zod.F("internal_note", zod.String().Optional()),
		zod.F("reason_id", zod.String().Optional()),
		zod.F("metadata", metadata().Optional()),
	))),
)

// AdminGetExchangeParams mirrors AdminGetExchangeParams.
var AdminGetExchangeParams = utils.CreateSelectParams()
