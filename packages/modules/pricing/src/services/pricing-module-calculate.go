package services

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

type sqlArgs struct{ args []any }

func (a *sqlArgs) add(v any) string {
	a.args = append(a.args, v)
	return fmt.Sprintf("$%d", len(a.args))
}

// repositoryCalculatePrices mirrors PricingRepository.calculatePrices.
func (s *PricingModuleService) repositoryCalculatePrices(ctx context.Context, priceSetIDs []any, pricingContext *dal.Record) ([]dal.Row, error) {
	c := dal.NewRecord()
	var quantity any
	var currency string
	if pricingContext != nil {
		for _, k := range pricingContext.Keys() {
			v, _ := pricingContext.Get(k)
			switch k {
			case "quantity":
				quantity = v
			case "currency_code":
				currency, _ = v.(string)
			default:
				c.Set(k, v)
			}
		}
	}
	if currency == "" {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Method calculatePrices requires currency_code in the pricing context")
	}
	var flattened []keyValue
	for _, kv := range flattenObjectToKeyValuePairs(c) {
		if arr, ok := kv.Value.([]any); ok {
			if len(arr) > 0 {
				flattened = append(flattened, kv)
			}
			continue
		}
		if isPresent(kv.Value) {
			flattened = append(flattened, kv)
		}
	}
	if len(flattened) > 10 {
		rows, err := dal.QueryRows(ctx, s.DB(), `SELECT DISTINCT attribute FROM (SELECT attribute FROM price_rule UNION ALL SELECT attribute FROM price_list_rule) as combined_rules_attributes`)
		if err != nil {
			return nil, err
		}
		available := map[string]bool{}
		for _, r := range rows {
			available[fmt.Sprint(r["attribute"])] = true
		}
		var kept []keyValue
		for _, kv := range flattened {
			if available[kv.Key] {
				kept = append(kept, kv)
			}
		}
		flattened = kept
	}
	a := &sqlArgs{}
	ids := make([]string, len(priceSetIDs))
	for i, id := range priceSetIDs {
		ids[i] = a.add(id)
	}
	if len(ids) == 0 {
		return []dal.Row{}, nil
	}
	var where []string
	where = append(where, fmt.Sprintf(`"price"."price_set_id" in (%s)`, strings.Join(ids, ", ")))
	where = append(where, fmt.Sprintf(`"price"."currency_code" = %s`, a.add(strings.ToLower(currency))))
	where = append(where, `"price"."deleted_at" is null`)
	if quantity != nil {
		q1, q2, q3, q4 := a.add(quantity), a.add(quantity), a.add(quantity), a.add(quantity)
		where = append(where, fmt.Sprintf(`(("price"."min_quantity" <= %s and "price"."max_quantity" >= %s) or ("price"."min_quantity" <= %s and "price"."max_quantity" is null) or ("price"."min_quantity" is null and "price"."max_quantity" is null) or ("price"."min_quantity" is null and "price"."max_quantity" >= %s))`, q1, q2, q3, q4))
	} else {
		where = append(where, `("price"."min_quantity" <= 1 or "price"."min_quantity" is null)`)
	}
	var joins string
	if len(flattened) > 0 {
		var plConds []string
		plArgs := func() string {
			for _, kv := range flattened {
				vals, isArr := kv.Value.([]any)
				if !isArr {
					vals = []any{kv.Value}
				}
				for _, v := range vals {
					b, _ := json.Marshal(v)
					plConds = append(plConds, fmt.Sprintf("(plr.attribute = %s AND plr.value @> %s)", a.add(kv.Key), a.add(string(b))))
				}
			}
			return strings.Join(plConds, " OR ")
		}
		status := a.add("active")
		plMatch := plArgs()
		joins += fmt.Sprintf(` left join (
            SELECT pl.id, pl.type, pl.rules_count
            FROM price_list pl
            WHERE pl.status = %s
              AND pl.deleted_at IS NULL
              AND (pl.starts_at IS NULL OR pl.starts_at <= now())
              AND (pl.ends_at IS NULL OR pl.ends_at >= now())
              AND (
                pl.rules_count = 0
                OR pl.rules_count = (
                  SELECT COUNT(*) FILTER (WHERE %s)
                  FROM price_list_rule plr
                  WHERE plr.price_list_id = pl.id
                    AND plr.deleted_at IS NULL
                )
              )
          ) as pl on "pl"."id" = "price"."price_list_id"`, status, plMatch)
		var prConds []string
		for _, kv := range flattened {
			if f, ok := kv.Value.(float64); ok {
				key := a.add(kv.Key)
				sv := a.add(strconv.FormatFloat(f, 'f', -1, 64))
				n1, n2, n3, n4 := a.add(f), a.add(f), a.add(f), a.add(f)
				prConds = append(prConds, fmt.Sprintf(`(pr.attribute = %s AND ((pr.operator = 'eq' AND pr.value = %s) OR (pr.operator = 'gt' AND %s > pr.value::numeric) OR (pr.operator = 'gte' AND %s >= pr.value::numeric) OR (pr.operator = 'lt' AND %s < pr.value::numeric) OR (pr.operator = 'lte' AND %s <= pr.value::numeric)))`, key, sv, n1, n2, n3, n4))
				continue
			}
			vals, isArr := kv.Value.([]any)
			if !isArr {
				vals = []any{kv.Value}
			}
			key := a.add(kv.Key)
			ph := make([]string, len(vals))
			for i, v := range vals {
				ph[i] = a.add(fmt.Sprint(v))
			}
			prConds = append(prConds, fmt.Sprintf("(pr.attribute = %s AND pr.value IN (%s))", key, strings.Join(ph, ",")))
		}
		joins += fmt.Sprintf(` left join LATERAL (
            SELECT
              COUNT(*) FILTER (WHERE %s) as matched_count,
              COUNT(*) as total_count
            FROM price_rule pr
            WHERE pr.price_id = price.id
              AND pr.deleted_at IS NULL
          ) pr_stats on true`, strings.Join(prConds, " OR "))
		where = append(where, `(("price"."price_list_id" is null and ("price"."rules_count" = 0 or pr_stats.matched_count = price.rules_count)) or ("price"."price_list_id" is not null and "pl"."id" is not null and ("price"."rules_count" = 0 or pr_stats.matched_count = price.rules_count)))`)
	} else {
		joins += fmt.Sprintf(` left join "price_list" as "pl" on "pl"."id" = "price"."price_list_id" and "pl"."status" = %s and ("pl"."deleted_at" is null) and ("pl"."starts_at" is null or "pl"."starts_at" <= CURRENT_TIMESTAMP) and ("pl"."ends_at" is null or "pl"."ends_at" >= CURRENT_TIMESTAMP)`, a.add("active"))
		where = append(where, `(("price"."price_list_id" is null and "price"."rules_count" = 0) or ("price"."price_list_id" is not null and "price"."rules_count" = 0 and "pl"."rules_count" = 0))`)
	}
	sql := `select "price"."id" as "id", "price"."price_set_id" as "price_set_id", "price"."amount" as "amount", "price"."raw_amount" as "raw_amount", "price"."min_quantity" as "min_quantity", "price"."max_quantity" as "max_quantity", "price"."currency_code" as "currency_code", "price"."price_list_id" as "price_list_id", "pl"."type" as "price_list_type", "price"."rules_count" as "rules_count", "pl"."rules_count" as "price_list_rules_count" from "price"` +
		joins + " where " + strings.Join(where, " and ") +
		` order by price.price_list_id IS NOT NULL DESC, price.rules_count + COALESCE(pl.rules_count, 0) DESC, "price"."amount" asc`
	return dal.QueryRows(ctx, s.DB(), sql, a.args...)
}

func isPresent(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case []any:
		return len(t) > 0
	case *dal.Record:
		return len(t.Keys()) > 0
	}
	return true
}

func rowStr(r dal.Row, k string) string {
	if r == nil || r[k] == nil {
		return ""
	}
	return fmt.Sprint(r[k])
}

func orNull(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func parseFloatOrNull(s string) any {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f == 0 {
		return nil
	}
	return f
}

func amountOrNull(r dal.Row) any {
	if r == nil || !isPresent(r["amount"]) {
		return nil
	}
	f, err := strconv.ParseFloat(fmt.Sprint(r["amount"]), 64)
	if err != nil {
		return nil
	}
	return f
}

func priceSummary(r dal.Row) *dal.Record {
	out := dal.NewRecord()
	out.Set("id", orNull(rowStr(r, "id")))
	out.Set("price_list_id", orNull(rowStr(r, "price_list_id")))
	out.Set("price_list_type", orNull(rowStr(r, "price_list_type")))
	out.Set("min_quantity", parseFloatOrNull(rowStr(r, "min_quantity")))
	out.Set("max_quantity", parseFloatOrNull(rowStr(r, "max_quantity")))
	return out
}

// isTaxInclusive mirrors isTaxInclusive.
func isTaxInclusive(rules []*dal.Record, preferences []*dal.Record, currencyCode string, regionID any) bool {
	find := func(list []*dal.Record, attr string, value any) *dal.Record {
		for _, r := range list {
			if get(r, "attribute") == attr && get(r, "value") == value {
				return r
			}
		}
		return nil
	}
	regionRule := find(rules, "region_id", regionID)
	regionPref := find(preferences, "region_id", regionID)
	currencyPref := find(preferences, "currency_code", currencyCode)
	if regionRule != nil && regionPref != nil {
		b, _ := get(regionPref, "is_tax_inclusive").(bool)
		return b
	}
	if currencyPref != nil {
		b, _ := get(currencyPref, "is_tax_inclusive").(bool)
		return b
	}
	return false
}

// CalculatePrices mirrors calculatePrices.
func (s *PricingModuleService) CalculatePrices(ctx context.Context, priceSetIDs []any, pricingContext *dal.Record) ([]*dal.Record, error) {
	rows, err := s.repositoryCalculatePrices(ctx, priceSetIDs, pricingContext)
	if err != nil {
		return nil, err
	}
	bySet := map[string][]dal.Row{}
	for _, r := range rows {
		k := rowStr(r, "price_set_id")
		bySet[k] = append(bySet[k], r)
	}
	type chosen struct{ calculated, original dal.Row }
	picked := map[string]chosen{}
	var priceIDs []any
	for setID, prices := range bySet {
		var priceListPrice, defaultPrice dal.Row
		for _, p := range prices {
			if priceListPrice == nil && rowStr(p, "price_list_id") != "" {
				priceListPrice = p
			}
			if defaultPrice == nil && rowStr(p, "price_list_id") == "" {
				defaultPrice = p
			}
		}
		if priceListPrice == nil && defaultPrice == nil {
			continue
		}
		var nonSalePL dal.Row
		for _, p := range prices {
			if rowStr(p, "price_list_type") != "sale" && rowStr(p, "price_list_id") != "" {
				nonSalePL = p
				break
			}
		}
		calculated := defaultPrice
		original := nonSalePL
		if original == nil {
			original = defaultPrice
		}
		if priceListPrice != nil {
			switch rowStr(priceListPrice, "price_list_type") {
			case "override":
				calculated, original = priceListPrice, priceListPrice
			case "sale":
				lowest := priceListPrice
				if original != nil && isPresent(original["amount"]) && isPresent(priceListPrice["amount"]) {
					if totals.Lte(priceListPrice["amount"], original["amount"]) {
						lowest = priceListPrice
					} else {
						lowest = original
					}
				}
				calculated = lowest
			}
		}
		picked[setID] = chosen{calculated, original}
		for _, p := range []dal.Row{calculated, original} {
			if p == nil {
				continue
			}
			id := p["id"]
			if !slices.Contains(priceIDs, id) {
				priceIDs = append(priceIDs, id)
			}
		}
	}
	rulesByPrice := map[string][]*dal.Record{}
	if priceIDs == nil {
		priceIDs = []any{}
	}
	rf := dal.NewRecord()
	rf.Set("price_id", priceIDs)
	rules, err := s.Internal("PriceRule").List(ctx, rf, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	for _, r := range rules {
		k := fmt.Sprint(get(r, "price_id"))
		rulesByPrice[k] = append(rulesByPrice[k], r)
	}
	var regionID any
	var preferences []*dal.Record
	if pricingContext != nil {
		var or []any
		for _, k := range pricingContext.Keys() {
			if k != "region_id" && k != "currency_code" {
				continue
			}
			v, _ := pricingContext.Get(k)
			if k == "region_id" {
				regionID = v
			}
			f := dal.NewRecord()
			f.Set("attribute", k)
			f.Set("value", v)
			or = append(or, f)
		}
		if len(or) > 0 {
			pf := dal.NewRecord()
			pf.Set("$or", or)
			preferences, err = s.Internal("PricePreference").List(ctx, pf, utilsmodulessdk.FindConfig{})
			if err != nil {
				return nil, err
			}
		}
	}
	var out []*dal.Record
	for _, id := range priceSetIDs {
		ch, ok := picked[fmt.Sprint(id)]
		if !ok {
			continue
		}
		calc, orig := ch.calculated, ch.original
		currency := rowStr(calc, "currency_code")
		r := dal.NewRecord()
		r.Set("id", id)
		r.Set("is_calculated_price_price_list", rowStr(calc, "price_list_id") != "")
		r.Set("is_calculated_price_tax_inclusive", isTaxInclusive(rulesByPrice[rowStr(calc, "id")], preferences, currency, regionID))
		r.Set("calculated_amount", amountOrNull(calc))
		if calc != nil && calc["raw_amount"] != nil {
			r.Set("raw_calculated_amount", calc["raw_amount"])
		} else {
			r.Set("raw_calculated_amount", nil)
		}
		r.Set("is_original_price_price_list", rowStr(orig, "price_list_id") != "")
		origInclusive := false
		if rowStr(orig, "id") != "" {
			oc := rowStr(orig, "currency_code")
			if oc == "" {
				oc = currency
			}
			origInclusive = isTaxInclusive(rulesByPrice[rowStr(orig, "id")], preferences, oc, regionID)
		}
		r.Set("is_original_price_tax_inclusive", origInclusive)
		r.Set("original_amount", amountOrNull(orig))
		if orig != nil && orig["raw_amount"] != nil {
			r.Set("raw_original_amount", orig["raw_amount"])
		} else {
			r.Set("raw_original_amount", nil)
		}
		r.Set("currency_code", orNull(currency))
		r.Set("calculated_price", priceSummary(calc))
		r.Set("original_price", priceSummary(orig))
		out = append(out, r)
	}
	if out == nil {
		out = []*dal.Record{}
	}
	return out, nil
}
