package orders

import (
	"math"
	"strconv"

	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

func idLike() *zod.Schema {
	return zod.Union(zod.String(), zod.Array(zod.String()), utils.CreateOperatorMap(nil, nil)).Optional()
}

// AdminGetOrdersParamsBase mirrors AdminGetOrdersParamsBase.
var AdminGetOrdersParamsBase = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(15), Offset: utils.Num(0)}).Merge(zod.Object(
	zod.F("id", idLike()),
	zod.F("status", idLike()),
	zod.F("sales_channel_id", zod.Array(zod.String()).Optional()),
	zod.F("region_id", utils.StringOrStrings().Optional()),
	zod.F("customer_id", utils.StringOrStrings().Optional()),
	zod.F("q", zod.String().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("total", utils.CreateOperatorMap(nil, nil).Optional()),
))

// adminGetOrdersParamsTransform mirrors AdminGetOrdersParamsTransform.
func adminGetOrdersParamsTransform(v any) (any, error) {
	in, ok := v.(*dal.Record)
	if !ok {
		return v, nil
	}
	out := dal.NewRecord()
	var total any
	for _, k := range in.Keys() {
		x, _ := in.Get(k)
		if k == "total" {
			total = x
			continue
		}
		out.Set(k, x)
	}
	if total != nil && total != "" {
		totals := dal.NewRecord()
		totals.Set("current_order_total", total)
		summary := dal.NewRecord()
		summary.Set("totals", totals)
		out.Set("summary", summary)
	}
	return out, nil
}

// AdminGetOrdersParams mirrors AdminGetOrdersParams (a transform: not made
// strict by zodValidator).
var AdminGetOrdersParams = AdminGetOrdersParamsBase.Transform(adminGetOrdersParamsTransform)

// AdminGetOrdersOrderParams mirrors AdminGetOrdersOrderParams.
var AdminGetOrdersOrderParams = utils.CreateSelectParams().Merge(zod.Object(
	zod.F("version", zod.Preprocess(func(v any) any {
		if s, ok := v.(string); ok && s != "" {
			return parseIntJS(s)
		}
		return v
	}, zod.Number().Optional())),
))

// parseIntJS mirrors parseInt(value) (NaN for non-numeric strings).
func parseIntJS(s string) any {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n') {
		i++
	}
	start := i
	if i < len(s) && (s[i] == '-' || s[i] == '+') {
		i++
	}
	digits := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == digits {
		return math.NaN()
	}
	f, _ := strconv.ParseFloat(s[start:i], 64)
	return f
}

var orderItem = zod.Object(
	zod.F("id", zod.String()),
	zod.F("quantity", zod.Number()),
)

// OrderCreateFulfillment mirrors OrderCreateFulfillment.
var OrderCreateFulfillment = zod.Object(
	zod.F("items", zod.Array(orderItem).Min(1)),
	zod.F("location_id", zod.String().Nullish()),
	zod.F("shipping_option_id", zod.String().Optional()),
	zod.F("no_notification", zod.Boolean().Optional()),
	zod.F("delivery_address", commonvalidators.AddressPayload().Optional()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

// AdminOrderCreateFulfillment mirrors AdminOrderCreateFulfillment.
var AdminOrderCreateFulfillment = utils.WithAdditionalData(OrderCreateFulfillment)

var label = zod.Object(
	zod.F("tracking_number", zod.String()),
	zod.F("tracking_url", commonvalidators.SafeHttpURL()),
	zod.F("label_url", commonvalidators.SafeHttpURL()),
)

// OrderCreateShipment mirrors OrderCreateShipment.
var OrderCreateShipment = zod.Object(
	zod.F("items", zod.Array(orderItem)),
	zod.F("labels", zod.Array(label).Optional()),
	zod.F("no_notification", zod.Boolean().Optional()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

// AdminOrderCreateShipment mirrors AdminOrderCreateShipment.
var AdminOrderCreateShipment = utils.WithAdditionalData(OrderCreateShipment)

// AdminMarkOrderFulfillmentAsDelivered mirrors AdminMarkOrderFulfillmentAsDelivered.
var AdminMarkOrderFulfillmentAsDelivered = zod.Object(zod.F("no_notification", zod.Boolean().Optional()))

// AdminOrderCancelFulfillment mirrors AdminOrderCancelFulfillment.
var AdminOrderCancelFulfillment = utils.WithAdditionalData(zod.Object(zod.F("no_notification", zod.Boolean().Optional())))

func versionParam() *zod.Schema {
	return zod.Preprocess(func(v any) any {
		if s, ok := v.(string); ok && s != "" {
			return parseIntJS(s)
		}
		return v
	}, zod.Number().Optional())
}

// AdminGetOrdersOrderItemsParams mirrors AdminGetOrdersOrderItemsParams.
var AdminGetOrdersOrderItemsParams = utils.CreateSelectParams().Merge(zod.Object(
	zod.F("id", zod.Union(zod.String(), zod.Array(zod.String())).Optional()),
	zod.F("item_id", zod.Union(zod.String(), zod.Array(zod.String())).Optional()),
	zod.F("version", versionParam()),
))

// AdminOrderChangesParams mirrors AdminOrderChangesParams.
var AdminOrderChangesParams = utils.CreateSelectParams().Merge(zod.Object(
	zod.F("id", zod.Union(zod.String(), zod.Array(zod.String())).Optional()),
	zod.F("status", zod.Union(zod.String(), zod.Array(zod.String())).Optional()),
	zod.F("change_type", zod.Union(zod.String(), zod.Array(zod.String())).Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
))

// AdminCompleteOrder mirrors AdminCompleteOrder.
var AdminCompleteOrder = utils.WithAdditionalData(zod.Object())

// AdminUpdateOrder mirrors AdminUpdateOrder.
var AdminUpdateOrder = zod.Object(
	zod.F("email", zod.String().Optional()),
	zod.F("shipping_address", commonvalidators.AddressPayload().Optional()),
	zod.F("billing_address", commonvalidators.AddressPayload().Optional()),
	zod.F("locale", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

// AdminCreateOrderCreditLines mirrors AdminCreateOrderCreditLines.
var AdminCreateOrderCreditLines = zod.Object(
	zod.F("amount", zod.Number()),
	zod.F("reference", zod.String()),
	zod.F("reference_id", zod.String()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

// AdminAuthorizeOrderPaymentSession mirrors AdminAuthorizeOrderPaymentSession.
var AdminAuthorizeOrderPaymentSession = zod.Object(zod.F("payment_session_id", zod.String()))

// AdminGetOrderShippingOptionList mirrors AdminGetOrderShippingOptionList.
var AdminGetOrderShippingOptionList = zod.Object()

// AdminTransferOrder mirrors AdminTransferOrder.
var AdminTransferOrder = zod.Object(
	zod.F("customer_id", zod.String()),
	zod.F("description", zod.String().Optional()),
	zod.F("internal_note", zod.String().Optional()),
	zod.F("update_order_email", zod.Boolean().Optional()),
)

// AdminTransferOrderToGuest mirrors AdminTransferOrderToGuest.
var AdminTransferOrderToGuest = zod.Object(
	zod.F("email", zod.String().Email()),
	zod.F("description", zod.String().Optional()),
	zod.F("internal_note", zod.String().Optional()),
)
