// Package utils mirrors @medusajs/core-flows/src/product/utils.
package utils

import "github.com/hitkey-io/medusago/packages/core/utils/src/dal"

// CreateDefaultInventoryItem mirrors createDefaultInventoryItem (undefined
// values are omitted, like MikroORM ignores them).
func CreateDefaultInventoryItem(variant *dal.Record) *dal.Record {
	out := dal.NewRecord()
	copyKey := func(to, from string) {
		if v, ok := variant.Get(from); ok {
			out.Set(to, v)
		}
	}
	copyKey("sku", "sku")
	copyKey("origin_country", "origin_country")
	copyKey("mid_code", "mid_code")
	copyKey("material", "material")
	copyKey("weight", "weight")
	copyKey("length", "length")
	copyKey("height", "height")
	copyKey("width", "width")
	copyKey("title", "title")
	copyKey("description", "title")
	copyKey("hs_code", "hs_code")
	out.Set("requires_shipping", true)
	return out
}
