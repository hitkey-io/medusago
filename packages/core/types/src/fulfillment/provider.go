// Package fulfillment mirrors the fulfillment provider types of
// @medusajs/types/src/fulfillment/provider.ts.
package fulfillment

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// IFulfillmentProvider mirrors IFulfillmentProvider. Records stand for the
// plain objects upstream passes.
type IFulfillmentProvider interface {
	// GetFulfillmentOptions returns the provider's fulfillment options.
	GetFulfillmentOptions(ctx context.Context) ([]any, error)
	// ValidateFulfillmentData returns the validated shipping method data.
	ValidateFulfillmentData(ctx context.Context, optionData, data, context_ *dal.Record) (any, error)
	// ValidateOption reports whether the option data is valid.
	ValidateOption(ctx context.Context, data *dal.Record) (bool, error)
	// CanCalculate reports whether the provider can calculate the price of
	// the shipping option.
	CanCalculate(ctx context.Context, data *dal.Record) (bool, error)
	// CalculatePrice returns { calculated_amount,
	// is_calculated_price_tax_inclusive }.
	CalculatePrice(ctx context.Context, optionData, data, context_ *dal.Record) (*dal.Record, error)
	// CreateFulfillment returns { data, labels }.
	CreateFulfillment(ctx context.Context, data *dal.Record, items []any, order *dal.Record, fulfillment *dal.Record) (*dal.Record, error)
	// CancelFulfillment cancels the fulfillment with its data.
	CancelFulfillment(ctx context.Context, data *dal.Record) (any, error)
	// CreateReturnFulfillment returns { data, labels }.
	CreateReturnFulfillment(ctx context.Context, fulfillment *dal.Record) (*dal.Record, error)
}
