package migrations

import "testing"

// Expected values computed with the upstream #hashStringToInt.
func TestHashStringToInt(t *testing.T) {
	cases := map[string]int32{
		"script_migrations:backfill-refund-status.js":           -354831399,
		"script_migrations:migrate-product-shipping-profile.js": -1710292647,
	}
	for in, want := range cases {
		if got := hashStringToInt(in); got != want {
			t.Errorf("hashStringToInt(%q) = %d, want %d", in, got, want)
		}
	}
}
