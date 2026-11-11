package utils

import "testing"

// Vectors of utils/__tests__/get-smallest-unit.ts.
func TestGetSmallestUnit(t *testing.T) {
	cases := []struct {
		amount   float64
		currency string
		want     int64
	}{
		{50098, "JPY", 50098},
		{5.124, "KWD", 5130},
		{2.675, "USD", 268},
		{100.54, "USD", 10054},
		{5.126, "KWD", 5130},
		{0.54, "USD", 54},
		{0.054, "USD", 5},
		{0.005104, "USD", 1},
		{0.004104, "USD", 0},
	}
	for _, c := range cases {
		if got := GetSmallestUnit(c.amount, c.currency); got != c.want {
			t.Errorf("GetSmallestUnit(%v, %s) = %d, want %d", c.amount, c.currency, got, c.want)
		}
	}
	if got := GetAmountFromSmallestUnit(float64(5130), "KWD"); got != 5.13 {
		t.Errorf("GetAmountFromSmallestUnit = %v", got)
	}
}
