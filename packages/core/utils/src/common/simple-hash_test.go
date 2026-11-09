package common

import (
	"strings"
	"testing"
)

// Expected values were produced by the upstream simpleHash/compressName.
func TestSimpleHash(t *testing.T) {
	cases := map[string]string{
		"cart_payment_collection":        "-4a39f6c9",
		"location_fulfillment_provider":  "-1e5992737",
		"product_variant_inventory_item": "17b4c4e35",
		"a":                              "2b606",
		"":                               "1505",
		"ünïcödé_tablé":                  "d7cf406f",
		strings.Repeat("x", 300):         "1aa3b3325",
	}
	for in, want := range cases {
		if got := SimpleHash(in); got != want {
			t.Errorf("SimpleHash(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCompressName(t *testing.T) {
	cases := map[string]string{
		"short_name": "short_name",
		"product_variant_price_set_product_variant_price_set_extra_long":         "prod_vari_pric_set_prod_vari_pric_set_extr_longc9c1f9a",
		"customerAccountHolderPaymentAccountHolderWithAVeryLongNameIndeed_xx_yy": "cust_acco_hold_paym_acco_hold_with_aver_long_name1e4c67168",
		strings.Repeat("a_a_a_b_b_c_", 6):                                        "a_b_c_a_b_c_a_b_c_a_b_c_a_b_c_a_b_c_314042b1d",
	}
	for in, want := range cases {
		if got := CompressName(in); got != want {
			t.Errorf("CompressName(%q) = %q, want %q", in, got, want)
		}
	}
}
