package utils

import "testing"

// Vectors of utils/__tests__/amounts.spec.ts.
func TestMinorUnits(t *testing.T) {
	to := []struct {
		amount   float64
		currency string
		want     float64
	}{
		{10, "usd", 1000}, {10.005, "eur", 1001}, {1000, "JPY", 1000}, {1000, "idr", 1000}, {10, "clp", 1000}, {1.234, "kwd", 1234},
	}
	for _, c := range to {
		if got := ToMinorUnits(c.amount, c.currency); got != c.want {
			t.Errorf("ToMinorUnits(%v, %s) = %v, want %v", c.amount, c.currency, got, c.want)
		}
	}
	from := []struct {
		amount   float64
		currency string
		want     float64
	}{
		{1000, "usd", 10}, {1000, "JPY", 1000}, {1234, "kwd", 1.234},
	}
	for _, c := range from {
		if got := FromMinorUnits(c.amount, c.currency); got != c.want {
			t.Errorf("FromMinorUnits(%v, %s) = %v, want %v", c.amount, c.currency, got, c.want)
		}
	}
}
