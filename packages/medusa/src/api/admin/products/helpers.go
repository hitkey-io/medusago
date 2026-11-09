package products

import (
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	productvariants "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-variants"
)

var isPricing = productvariants.IsPricing

// RemapKeysForProduct mirrors remapKeysForProduct.
func RemapKeysForProduct(fields []string) []string {
	var product, pricing []string
	for _, f := range fields {
		if isPricing(f) {
			pricing = append(pricing, strings.Replace(f, "variants.prices.", "variants.price_set.prices.", 1))
		} else {
			product = append(product, f)
		}
	}
	return append(product, pricing...)
}

// RemapKeysForVariant mirrors remapKeysForVariant.
var RemapKeysForVariant = productvariants.RemapKeysForVariant

var records = productvariants.Records

// RemapProductResponse mirrors remapProductResponse.
func RemapProductResponse(p any) any {
	r, ok := p.(*dal.Record)
	if !ok || r == nil {
		return p
	}
	out := dal.NewRecord()
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		out.Set(k, v)
	}
	if v, ok := r.Get("variants"); ok {
		if list, ok := records(v); ok {
			mapped := make([]any, len(list))
			for i, x := range list {
				mapped[i] = RemapVariantResponse(x)
			}
			out.Set("variants", mapped)
		} else {
			// product.variants?.map(...) on null: undefined, dropped by JSON
			out.Delete("variants")
		}
	}
	return out
}

// RemapVariantResponse mirrors remapVariantResponse.
var RemapVariantResponse = productvariants.RemapVariantResponse
