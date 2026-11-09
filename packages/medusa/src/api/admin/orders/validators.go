package orders

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
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
