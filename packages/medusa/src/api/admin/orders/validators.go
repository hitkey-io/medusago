package orders

import (
	"math"
	"strconv"

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
