// Package computeactions mirrors @medusajs/promotion/src/utils/compute-actions.
package computeactions

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

// attributeValues keeps the distinct values of an attribute in insertion
// order (a JS Set: "1" and 1 are different values).
type attributeValues struct {
	values []any
	seen   map[string]bool
}

func (a *attributeValues) add(v any) {
	k := fmt.Sprintf("%T:%v", v, v)
	if a.seen[k] {
		return
	}
	a.seen[k] = true
	a.values = append(a.values, v)
}

// plain converts typed record lists to []any, like the JSON shape the
// flattening works on.
func plain(v any) any {
	switch t := v.(type) {
	case []*dal.Record:
		out := make([]any, len(t))
		for i, r := range t {
			out[i] = plain(r)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = plain(x)
		}
		return out
	case *dal.Record:
		if t == nil {
			return nil
		}
		out := dal.NewRecord()
		for _, k := range t.Keys() {
			x, _ := t.Get(k)
			out.Set(k, plain(x))
		}
		return out
	}
	return v
}

// jsString is String(v) / v.toString().
func jsString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return totals.JSNumberToString(t)
	case float32:
		return totals.JSNumberToString(float64(t))
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case []any:
		parts := make([]string, len(t))
		for i, x := range t {
			if x != nil {
				parts[i] = jsString(x)
			}
		}
		return strings.Join(parts, ",")
	case *dal.Record:
		return "[object Object]"
	}
	return fmt.Sprint(v)
}

// jsNumber is Number(v): ok is false for NaN.
func jsNumber(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, !math.IsNaN(t)
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case bool:
		if t {
			return 1, true
		}
		return 0, true
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0, true
		}
		if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
			n, err := strconv.ParseInt(s[2:], 16, 64)
			return float64(n), err == nil
		}
		if s == "Infinity" || s == "+Infinity" {
			return math.Inf(1), true
		}
		if s == "-Infinity" {
			return math.Inf(-1), true
		}
		if strings.ContainsAny(s, "_") || strings.EqualFold(s, "inf") || strings.EqualFold(s, "+inf") || strings.EqualFold(s, "-inf") || strings.EqualFold(s, "nan") {
			return 0, false
		}
		f, err := strconv.ParseFloat(s, 64)
		return f, err == nil
	case []any:
		if len(t) == 0 {
			return 0, true
		}
		if len(t) == 1 {
			return jsNumber(jsString(t[0]))
		}
		return 0, false
	}
	return 0, false
}

func quoteSQL(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// BuildPromotionRuleQueryFilterFromContext mirrors
// buildPromotionRuleQueryFilterFromContext: a raw filter excluding the
// promotions with a rule (promotion, target or buy rule) the context cannot
// satisfy. ok is false when the condition would be too long (null upstream).
func BuildPromotionRuleQueryFilterFromContext(ctx context.Context, db dal.Querier, applicationContext *dal.Record) (filter dal.RawFilter, ok bool, err error) {
	var items, methods any = []any{}, []any{}
	rest := dal.NewRecord()
	if applicationContext != nil {
		for _, k := range applicationContext.Keys() {
			v, _ := applicationContext.Get(k)
			switch k {
			case "items":
				if v != nil {
					items = v
				}
			case "shipping_methods":
				if v != nil {
					methods = v
				}
			default:
				rest.Set(k, v)
			}
		}
	}

	var order []string
	attributes := map[string]*attributeValues{}
	addAll := func(prefix string, pairs []keyValue) {
		for _, kv := range pairs {
			key := prefix + kv.Key
			if attributes[key] == nil {
				attributes[key] = &attributeValues{seen: map[string]bool{}}
				order = append(order, key)
			}
			if arr, isArr := kv.Value.([]any); isArr {
				for _, v := range arr {
					attributes[key].add(v)
				}
			} else {
				attributes[key].add(kv.Value)
			}
		}
	}
	addAll("items.", flattenObjectToKeyValuePairs(plain(items)))
	addAll("shipping_methods.", flattenObjectToKeyValuePairs(plain(methods)))
	addAll("", flattenObjectToKeyValuePairs(plain(rest)))

	if len(order) > 10 {
		rows, err := dal.QueryRows(ctx, db, `
        SELECT DISTINCT attribute
        FROM promotion_rule
        WHERE deleted_at IS NULL
      `)
		if err != nil {
			return nil, false, err
		}
		available := map[string]bool{}
		for _, r := range rows {
			available[fmt.Sprint(r["attribute"])] = true
		}
		kept := order[:0]
		for _, a := range order {
			if available[a] {
				kept = append(kept, a)
			} else {
				delete(attributes, a)
			}
		}
		order = kept
	}

	var conditions []string
	for _, attribute := range order {
		values := attributes[attribute].values
		strs := make([]string, len(values))
		var numbers []float64
		for i, v := range values {
			strs[i] = quoteSQL(jsString(v))
			if n, isNum := jsNumber(v); isNum {
				numbers = append(numbers, n)
			}
		}
		stringValues := strings.Join(strs, ",")
		escaped := quoteSQL(attribute)
		if len(stringValues) > 0 {
			conditions = append(conditions, fmt.Sprintf(`(pr.attribute = %s AND pr.operator IN ('in', 'eq') AND NOT EXISTS (
        SELECT 1
        FROM promotion_rule_value prv_inner
        WHERE prv_inner.promotion_rule_id = pr.id
          AND prv_inner.deleted_at IS NULL
          AND prv_inner.value IN (%s)
      ))`, escaped, stringValues))
		}
		if len(numbers) > 0 {
			minValue, maxValue := math.Inf(1), math.Inf(-1)
			for _, n := range numbers {
				minValue = math.Min(minValue, n)
				maxValue = math.Max(maxValue, n)
			}
			minS, maxS := totals.JSNumberToString(minValue), totals.JSNumberToString(maxValue)
			conditions = append(conditions,
				fmt.Sprintf(`(pr.attribute = %s AND pr.operator = 'gt' AND CAST(prv.value AS DECIMAL) >= %s)`, escaped, maxS),
				fmt.Sprintf(`(pr.attribute = %s AND pr.operator = 'gte' AND prv.value NOT IN (%s) AND CAST(prv.value AS DECIMAL) > %s)`, escaped, stringValues, maxS),
				fmt.Sprintf(`(pr.attribute = %s AND pr.operator = 'lt' AND CAST(prv.value AS DECIMAL) <= %s)`, escaped, minS),
				fmt.Sprintf(`(pr.attribute = %s AND pr.operator = 'lte' AND prv.value NOT IN (%s) AND CAST(prv.value AS DECIMAL) < %s)`, escaped, stringValues, minS),
			)
		}
	}

	if len(order) == 0 {
		return func(alias string, _ func(any) string) string {
			return excludePromotionsWithRuleMatching(alias, "TRUE")
		}, true, nil
	}

	joined := strings.Join(conditions, " OR ")
	if float64(len(joined)) > 2147483648*0.9 {
		return nil, false, nil
	}
	keys := make([]string, len(order))
	for i, a := range order {
		keys[i] = quoteSQL(a)
	}
	attributeKeys := strings.Join(keys, ",")
	unsatisfiable := fmt.Sprintf(`(
    pr.attribute NOT IN (%s)
    OR (pr.attribute IN (%s) AND (%s))
  )`, attributeKeys, attributeKeys, joined)
	return func(alias string, _ func(any) string) string {
		return excludePromotionsWithRuleMatching(alias, unsatisfiable)
	}, true, nil
}

// ruleSources mirrors RULE_SOURCES: the promotion rules, the target rules and
// the buy rules of a promotion.
var ruleSources = []struct{ from, promotionID string }{
	{`promotion_promotion_rule ppr
      JOIN promotion_rule pr ON ppr.promotion_rule_id = pr.id`, "ppr.promotion_id"},
	{`promotion_application_method am
      JOIN application_method_target_rules amtr ON am.id = amtr.application_method_id
        AND am.deleted_at IS NULL
      JOIN promotion_rule pr ON amtr.promotion_rule_id = pr.id`, "am.promotion_id"},
	{`promotion_application_method am
      JOIN application_method_buy_rules ambr ON am.id = ambr.application_method_id
        AND am.deleted_at IS NULL
      JOIN promotion_rule pr ON ambr.promotion_rule_id = pr.id`, "am.promotion_id"},
}

// excludePromotionsWithRuleMatching mirrors excludePromotionsWithRuleMatching.
func excludePromotionsWithRuleMatching(alias, condition string) string {
	parts := make([]string, len(ruleSources))
	for i, src := range ruleSources {
		parts[i] = fmt.Sprintf(`NOT EXISTS (
      SELECT 1
      FROM %s
      LEFT JOIN promotion_rule_value prv
        ON prv.promotion_rule_id = pr.id AND prv.deleted_at IS NULL
      WHERE %s = %s.id AND %s
    )`, src.from, src.promotionID, alias, condition)
	}
	return strings.Join(parts, " AND ")
}
