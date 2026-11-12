// Package providers mirrors @medusajs/tax/src/providers.
package providers

import (
	"context"

	taxtypes "github.com/hitkey-io/medusago/packages/core/types/src/tax"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// SystemTaxIdentifier mirrors SystemTaxService.identifier.
const SystemTaxIdentifier = "system"

// SystemTaxService mirrors SystemTaxService: one tax line per rate.
type SystemTaxService struct{}

// GetIdentifier mirrors getIdentifier.
func (SystemTaxService) GetIdentifier() string { return SystemTaxIdentifier }

func taxLine(rate *dal.Record, key string, id any) *dal.Record {
	get := func(k string) any { v, _ := rate.Get(k); return v }
	l := dal.NewRecord()
	l.Set("rate_id", get("id"))
	r := get("rate")
	if r == nil || r == float64(0) {
		r = float64(0)
	}
	l.Set("rate", r)
	l.Set("name", get("name"))
	l.Set("code", get("code"))
	l.Set(key, id)
	l.Set("provider_id", SystemTaxIdentifier)
	return l
}

// GetTaxLines mirrors getTaxLines.
func (SystemTaxService) GetTaxLines(_ context.Context, itemLines []taxtypes.ItemTaxCalculationLine, shippingLines []taxtypes.ShippingTaxCalculationLine, _ *dal.Record) ([]*dal.Record, error) {
	out := []*dal.Record{}
	for _, l := range itemLines {
		id, _ := l.LineItem.Get("id")
		for _, r := range l.Rates {
			out = append(out, taxLine(r, "line_item_id", id))
		}
	}
	for _, l := range shippingLines {
		id, _ := l.ShippingLine.Get("id")
		for _, r := range l.Rates {
			out = append(out, taxLine(r, "shipping_line_id", id))
		}
	}
	return out, nil
}
