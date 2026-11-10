// Package helpers mirrors @medusajs/core-flows/src/product/helpers.
package helpers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/csv"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func records(v any) []*dal.Record {
	var out []*dal.Record
	switch t := v.(type) {
	case []any:
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
	case []*dal.Record:
		out = t
	}
	return out
}

// BeautifyKey mirrors beautifyKey: snake_case -> "Title Case".
func BeautifyKey(key string) string {
	parts := strings.Split(key, "_")
	for i, p := range parts {
		if p != "" {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

func prefixFields(r *dal.Record, out *dal.Record, prefix string) {
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		out.Set(BeautifyKey(prefix+"_"+k), v)
	}
}

// LocaleCompare approximates String.prototype.localeCompare.
func LocaleCompare(a, b string) int {
	if c := strings.Compare(strings.ToLower(a), strings.ToLower(b)); c != 0 {
		return c
	}
	return -strings.Compare(a, b)
}

// NormalizeForExport mirrors normalizeForExport: one row per variant (or
// one per product without variants).
func NormalizeForExport(products []any, regions []*dal.Record) ([]any, error) {
	regionsMap := map[any]*dal.Record{}
	for _, r := range regions {
		regionsMap[get(r, "id")] = r
	}
	out := []any{}
	for _, p := range products {
		product, _ := p.(*dal.Record)
		variants := records(get(product, "variants"))
		if len(variants) == 0 {
			out = append(out, normalizeProductForExport(product))
			continue
		}
		for _, v := range variants {
			row := normalizeProductForExport(product)
			vr, err := normalizeVariantForExport(v, regionsMap, product)
			if err != nil {
				return nil, err
			}
			for _, k := range vr.Keys() {
				x, _ := vr.Get(k)
				row.Set(k, x)
			}
			row.Delete("Product Variants")
			out = append(out, row)
		}
	}
	return out, nil
}

func flattenIndexed(out *dal.Record, list any, prefix string, value func(*dal.Record) any) {
	for i, r := range records(list) {
		out.Set(BeautifyKey(fmt.Sprintf("%s_%d", prefix, i+1)), value(r))
	}
}

func normalizeProductForExport(product *dal.Record) *dal.Record {
	res := dal.NewRecord()
	prefixFields(product, res, "product")
	flattenIndexed(res, get(product, "images"), "product_image", func(r *dal.Record) any { return get(r, "url") })
	flattenIndexed(res, get(product, "tags"), "product_tag", func(r *dal.Record) any { return get(r, "value") })
	flattenIndexed(res, get(product, "sales_channels"), "product_sales_channel", func(r *dal.Record) any { return get(r, "id") })
	flattenIndexed(res, get(product, "categories"), "product_category", func(r *dal.Record) any { return get(r, "id") })
	for _, k := range []string{"Product Images", "Product Tags", "Product Sales Channels", "Product Categories", "Product Metadata", "Product Type", "Product Collection", "Product Options"} {
		res.Delete(k)
	}
	return res
}

func normalizeVariantForExport(variant *dal.Record, regionsMap map[any]*dal.Record, product *dal.Record) (*dal.Record, error) {
	prices := dal.NewRecord()
	if priceSet, ok := get(variant, "price_set").(*dal.Record); ok {
		list := records(get(priceSet, "prices"))
		sorted := append([]*dal.Record{}, list...)
		sort.SliceStable(sorted, func(i, j int) bool {
			return LocaleCompare(fmt.Sprint(get(sorted[j], "currency_code")), fmt.Sprint(get(sorted[i], "currency_code"))) < 0
		})
		for _, price := range sorted {
			rules := records(get(price, "price_rules"))
			var regionRule *dal.Record
			for _, r := range rules {
				if get(r, "attribute") == "region_id" {
					regionRule = r
					break
				}
			}
			if regionRule != nil {
				region := regionsMap[get(regionRule, "value")]
				if region == nil {
					return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Region with id %v not found", get(regionRule, "value")))
				}
				name := strings.Join(strings.Split(strings.ToLower(fmt.Sprint(get(region, "name"))), " "), "_")
				key := fmt.Sprintf("variant_price_%s_[%s]", name, strings.ToUpper(fmt.Sprint(get(region, "currency_code"))))
				prices.Set(BeautifyKey(key), get(price, "amount"))
			} else if len(rules) == 0 {
				prices.Set(BeautifyKey("variant_price_"+strings.ToUpper(fmt.Sprint(get(price, "currency_code")))), get(price, "amount"))
			}
		}
	}
	productOptions := records(get(product, "options"))
	res := dal.NewRecord()
	prefixFields(variant, res, "variant")
	for _, k := range prices.Keys() {
		v, _ := prices.Get(k)
		res.Set(k, v)
	}
	for i, option := range records(get(variant, "options")) {
		var prodOption *dal.Record
		for _, po := range productOptions {
			if get(po, "id") == get(option, "option_id") {
				prodOption = po
				break
			}
		}
		idKey := BeautifyKey(fmt.Sprintf("variant_option_%d_id", i+1))
		nameKey := BeautifyKey(fmt.Sprintf("variant_option_%d_name", i+1))
		if prodOption != nil {
			res.Set(idKey, get(prodOption, "id"))
			res.Set(nameKey, get(prodOption, "title"))
		} else {
			res.Set(idKey, csv.Undefined)
			res.Set(nameKey, csv.Undefined)
		}
		res.Set(BeautifyKey(fmt.Sprintf("variant_option_%d_value", i+1)), get(option, "value"))
	}
	flattenIndexed(res, get(variant, "images"), "variant_image", func(r *dal.Record) any { return get(r, "url") })
	res.Delete("Variant Price Set")
	res.Delete("Variant Options")
	res.Delete("Variant Images")
	return res, nil
}

// ProductExportOptions are the deeks / json2csv options of the product
// export.
var ProductExportOptions = csv.DeeksOptions{ArrayIndexesAsKeys: true, ExpandNestedObjects: true, ExpandArrayObjects: true, IgnoreEmptyArraysWhenExpanding: true, EscapeNestedDots: true}

// AppendProductExportKeys mirrors appendProductExportKeys: the keys in
// discovery order, without duplicates.
func AppendProductExportKeys(normalized []any, seen *[]string, seenSet map[string]bool) {
	for _, keys := range csv.DeepKeysFromList(normalized, ProductExportOptions) {
		for _, k := range keys {
			if !seenSet[k] {
				seenSet[k] = true
				*seen = append(*seen, k)
			}
		}
	}
}
