package claims

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/returns"
)

func metadata() *zod.Schema { return zod.Record(zod.String(), zod.Unknown()) }

// AdminGetOrdersOrderParams mirrors AdminGetOrdersOrderParams (claims).
var AdminGetOrdersOrderParams = returns.AdminGetOrdersOrderParams

// AdminGetOrdersParams mirrors AdminGetOrdersParams (claims).
var AdminGetOrdersParams = returns.AdminGetOrdersParams

// AdminPostOrderClaimsReqSchema mirrors AdminPostOrderClaimsReqSchema.
var AdminPostOrderClaimsReqSchema = zod.Object(
	zod.F("type", zod.Enum("refund", "replace")),
	zod.F("order_id", zod.String()),
	zod.F("description", zod.String().Optional()),
	zod.F("internal_note", zod.String().Optional()),
	zod.F("reason_id", zod.String().Nullish()),
	zod.F("metadata", metadata().Nullish()),
)

// AdminPostClaimsShippingReqSchema mirrors AdminPostClaimsShippingReqSchema.
var AdminPostClaimsShippingReqSchema = zod.Object(
	zod.F("shipping_option_id", zod.String()),
	zod.F("custom_amount", zod.Number().Optional()),
	zod.F("description", zod.String().Optional()),
	zod.F("internal_note", zod.String().Optional()),
	zod.F("metadata", metadata().Optional()),
)

// AdminPostClaimsAddItemsReqSchema mirrors AdminPostClaimsAddItemsReqSchema.
var AdminPostClaimsAddItemsReqSchema = zod.Object(
	zod.F("items", zod.Array(zod.Object(
		zod.F("variant_id", zod.String()),
		zod.F("quantity", zod.Number()),
		zod.F("unit_price", zod.Number().Optional()),
		zod.F("internal_note", zod.String().Optional()),
		zod.F("metadata", metadata().Optional()),
	))),
)

// AdminPostClaimsRequestReturnItemsReqSchema mirrors AdminPostClaimsRequestReturnItemsReqSchema.
var AdminPostClaimsRequestReturnItemsReqSchema = zod.Object(
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

// AdminPostClaimItemsReqSchema mirrors AdminPostClaimItemsReqSchema.
var AdminPostClaimItemsReqSchema = zod.Object(
	zod.F("items", zod.Array(zod.Object(
		zod.F("id", zod.String()),
		zod.F("quantity", zod.Number()),
		zod.F("reason", zod.Enum("missing_item", "wrong_item", "production_failure", "other").Optional()),
		zod.F("description", zod.String().Optional()),
		zod.F("internal_note", zod.String().Optional()),
		zod.F("metadata", metadata().Nullish()),
	))),
)
