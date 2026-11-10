package fulfillments

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// AdminFulfillmentParams mirrors AdminFulfillmentParams.
var AdminFulfillmentParams = utils.CreateSelectParams()

var adminCreateFulfillmentItem = zod.Object(
	zod.F("title", zod.String()),
	zod.F("sku", zod.String()),
	zod.F("quantity", zod.Number()),
	zod.F("barcode", zod.String()),
	zod.F("line_item_id", zod.String().Nullish()),
	zod.F("inventory_item_id", zod.String().Nullish()),
)

var adminCreateFulfillmentLabel = zod.Object(
	zod.F("tracking_number", zod.String()),
	zod.F("tracking_url", commonvalidators.SafeHttpURL()),
	zod.F("label_url", commonvalidators.SafeHttpURL()),
)

// AdminCreateFulfillment mirrors AdminCreateFulfillment.
var AdminCreateFulfillment = zod.Object(
	zod.F("location_id", zod.String()),
	zod.F("provider_id", zod.String()),
	zod.F("delivery_address", commonvalidators.AddressPayload()),
	zod.F("items", zod.Array(adminCreateFulfillmentItem)),
	zod.F("labels", zod.Array(adminCreateFulfillmentLabel)),
	zod.F("order_id", zod.String()),
	zod.F("shipping_option_id", zod.String().Nullish()),
	zod.F("data", zod.Record(zod.String(), zod.Unknown()).Nullable()),
	zod.F("packed_at", zod.CoerceDate().Nullish()),
	zod.F("shipped_at", zod.CoerceDate().Nullish()),
	zod.F("delivered_at", zod.CoerceDate().Nullish()),
	zod.F("canceled_at", zod.CoerceDate().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullable()),
)

// AdminCreateShipment mirrors AdminCreateShipment.
var AdminCreateShipment = zod.Object(
	zod.F("labels", zod.Array(adminCreateFulfillmentLabel)),
)
