package returns

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

func stringOrStrings() *zod.Schema {
	return zod.Union(zod.String(), zod.Array(zod.String())).Optional()
}

func metadata() *zod.Schema { return zod.Record(zod.String(), zod.Unknown()) }

// AdminGetOrdersOrderParams mirrors AdminGetOrdersOrderParams.
var AdminGetOrdersOrderParams = utils.CreateSelectParams().Merge(zod.Object(
	zod.F("id", stringOrStrings()),
	zod.F("status", stringOrStrings()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
))

// AdminGetReturnParams mirrors AdminGetReturnParams.
var AdminGetReturnParams = utils.CreateSelectParams()

// AdminGetOrdersParams mirrors AdminGetOrdersParams (returns).
var AdminGetOrdersParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(15), Offset: utils.Num(0)}).Merge(zod.Object(
	zod.F("id", stringOrStrings()),
	zod.F("order_id", stringOrStrings()),
	zod.F("status", stringOrStrings()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
))

// AdminPostReturnsReqSchema mirrors AdminPostReturnsReqSchema.
var AdminPostReturnsReqSchema = zod.Object(
	zod.F("order_id", zod.String()),
	zod.F("location_id", zod.String().Optional()),
	zod.F("description", zod.String().Optional()),
	zod.F("internal_note", zod.String().Optional()),
	zod.F("no_notification", zod.Boolean().Optional()),
	zod.F("metadata", metadata().Nullish()),
)

// AdminPostReceiveReturnsReqSchema mirrors AdminPostReceiveReturnsReqSchema.
var AdminPostReceiveReturnsReqSchema = zod.Object(
	zod.F("internal_note", zod.String().Optional()),
	zod.F("description", zod.String().Optional()),
	zod.F("metadata", metadata().Nullish()),
)

// AdminPostReceiveReturnItemsReqSchema mirrors AdminPostReceiveReturnItemsReqSchema.
var AdminPostReceiveReturnItemsReqSchema = zod.Object(
	zod.F("items", zod.Array(zod.Object(
		zod.F("id", zod.String()),
		zod.F("quantity", zod.Number().Min(1)),
		zod.F("internal_note", zod.String().Optional()),
	))),
)

// AdminPostReturnsShippingReqSchema mirrors AdminPostReturnsShippingReqSchema.
var AdminPostReturnsShippingReqSchema = zod.Object(
	zod.F("shipping_option_id", zod.String()),
	zod.F("custom_amount", zod.Number().Optional()),
	zod.F("description", zod.String().Optional()),
	zod.F("internal_note", zod.String().Optional()),
	zod.F("metadata", metadata().Optional()),
)

// AdminPostReturnsRequestItemsReqSchema mirrors AdminPostReturnsRequestItemsReqSchema.
var AdminPostReturnsRequestItemsReqSchema = zod.Object(
	zod.F("items", zod.Array(zod.Object(
		zod.F("id", zod.String()),
		zod.F("quantity", zod.Number()),
		zod.F("description", zod.String().Optional()),
		zod.F("internal_note", zod.String().Optional()),
		zod.F("reason_id", zod.String().Optional()),
		zod.F("metadata", metadata().Optional()),
	))),
)

// AdminPostReturnsConfirmRequestReqSchema mirrors AdminPostReturnsConfirmRequestReqSchema.
var AdminPostReturnsConfirmRequestReqSchema = zod.Object(
	zod.F("no_notification", zod.Boolean().Optional()),
)

// AdminPostReturnsReturnReqSchema mirrors AdminPostReturnsReturnReqSchema.
var AdminPostReturnsReturnReqSchema = zod.Object(
	zod.F("location_id", zod.String().Optional()),
	zod.F("no_notification", zod.Boolean().Optional()),
	zod.F("metadata", metadata().Nullish()),
)

// AdminPostCancelReturnReqSchema mirrors AdminPostCancelReturnReqSchema.
var AdminPostCancelReturnReqSchema = zod.Object(
	zod.F("no_notification", zod.Boolean().Optional()),
)

// AdminPostReturnsShippingActionReqSchema mirrors AdminPostReturnsShippingActionReqSchema.
var AdminPostReturnsShippingActionReqSchema = zod.Object(
	zod.F("custom_amount", zod.Number().Nullish().Optional()),
	zod.F("internal_note", zod.String().Nullish().Optional()),
	zod.F("metadata", metadata().Nullish().Optional()),
)

// AdminPostReturnsRequestItemsActionReqSchema mirrors AdminPostReturnsRequestItemsActionReqSchema.
var AdminPostReturnsRequestItemsActionReqSchema = zod.Object(
	zod.F("quantity", zod.Number().Optional()),
	zod.F("internal_note", zod.String().Nullish().Optional()),
	zod.F("reason_id", zod.String().Nullish().Optional()),
	zod.F("metadata", metadata().Nullish().Optional()),
)
