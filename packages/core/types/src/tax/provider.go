// Package tax mirrors the tax provider types of
// @medusajs/types/src/tax/provider.ts.
package tax

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// ItemTaxCalculationLine mirrors ItemTaxCalculationLine: a taxable line item
// ({ id, product_id, product_type_id, quantity, ... }) with its tax rates.
type ItemTaxCalculationLine struct {
	LineItem *dal.Record
	Rates    []*dal.Record
}

// ShippingTaxCalculationLine mirrors ShippingTaxCalculationLine: a taxable
// shipping line ({ id, shipping_option_id, ... }) with its tax rates.
type ShippingTaxCalculationLine struct {
	ShippingLine *dal.Record
	Rates        []*dal.Record
}

// ITaxProvider mirrors ITaxProvider. GetTaxLines returns item tax lines
// ({ rate_id, rate, name, code, line_item_id, provider_id }) and shipping
// tax lines ({ ..., shipping_line_id }); context is the tax calculation
// context ({ address, customer, is_return, ... }).
type ITaxProvider interface {
	GetIdentifier() string
	GetTaxLines(ctx context.Context, itemLines []ItemTaxCalculationLine, shippingLines []ShippingTaxCalculationLine, context_ *dal.Record) ([]*dal.Record, error)
}
