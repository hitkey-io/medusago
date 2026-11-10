// Package orderedits mirrors api/admin/order-edits.
package orderedits

import "github.com/hitkey-io/medusago/packages/core/framework/src/zod"

func metadata() *zod.Schema { return zod.Record(zod.String(), zod.Unknown()) }

// AdminPostOrderEditsReqSchema mirrors AdminPostOrderEditsReqSchema.
var AdminPostOrderEditsReqSchema = zod.Object(
	zod.F("order_id", zod.String()),
	zod.F("description", zod.String().Optional()),
	zod.F("internal_note", zod.String().Optional()),
	zod.F("metadata", metadata().Nullish()),
)

// AdminPostOrderEditsRequestReqSchema mirrors AdminPostOrderEditsRequestReqSchema.
var AdminPostOrderEditsRequestReqSchema = zod.Object(zod.F("no_notification", zod.Boolean().Optional()))

// AdminPostOrderEditsShippingReqSchema mirrors AdminPostOrderEditsShippingReqSchema.
var AdminPostOrderEditsShippingReqSchema = zod.Object(
	zod.F("shipping_option_id", zod.String()),
	zod.F("custom_amount", zod.Number().Optional()),
	zod.F("description", zod.String().Optional()),
	zod.F("internal_note", zod.String().Optional()),
	zod.F("metadata", metadata().Optional()),
)

// AdminPostOrderEditsAddItemsReqSchema mirrors AdminPostOrderEditsAddItemsReqSchema.
var AdminPostOrderEditsAddItemsReqSchema = zod.Object(
	zod.F("items", zod.Array(zod.Object(
		zod.F("variant_id", zod.String()),
		zod.F("quantity", zod.Number()),
		zod.F("unit_price", zod.Number().Nullish()),
		zod.F("compare_at_unit_price", zod.Number().Nullish()),
		zod.F("internal_note", zod.String().Nullish()),
		zod.F("allow_backorder", zod.Boolean().Optional()),
		zod.F("metadata", metadata().Optional()),
	))),
)

// AdminPostOrderEditsUpdateItemQuantityReqSchema mirrors AdminPostOrderEditsUpdateItemQuantityReqSchema.
var AdminPostOrderEditsUpdateItemQuantityReqSchema = zod.Object(
	zod.F("quantity", zod.Number()),
	zod.F("unit_price", zod.Number().Nullish()),
	zod.F("compare_at_unit_price", zod.Number().Nullish()),
	zod.F("internal_note", zod.String().Nullish().Optional()),
	zod.F("metadata", metadata().Nullish()),
)

// AdminPostOrderEditsShippingActionReqSchema mirrors AdminPostOrderEditsShippingActionReqSchema.
var AdminPostOrderEditsShippingActionReqSchema = zod.Object(
	zod.F("custom_amount", zod.Number().Nullish().Optional()),
	zod.F("internal_note", zod.String().Nullish().Optional()),
	zod.F("metadata", metadata().Nullish().Optional()),
)

// AdminPostOrderEditsItemsActionReqSchema mirrors AdminPostOrderEditsItemsActionReqSchema.
var AdminPostOrderEditsItemsActionReqSchema = zod.Object(
	zod.F("quantity", zod.Number().Optional()),
	zod.F("unit_price", zod.Number().Nullish()),
	zod.F("compare_at_unit_price", zod.Number().Nullish()),
	zod.F("internal_note", zod.String().Nullish().Optional()),
)
