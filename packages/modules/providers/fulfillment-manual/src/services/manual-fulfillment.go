// Package services mirrors @medusajs/fulfillment-manual/src/services.
package services

import (
	"context"
	"errors"

	fulfillmenttypes "github.com/hitkey-io/medusago/packages/core/types/src/fulfillment"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Identifier mirrors ManualFulfillmentService.identifier.
const Identifier = "manual"

// ManualFulfillmentService mirrors ManualFulfillmentService.
type ManualFulfillmentService struct{}

var _ fulfillmenttypes.IFulfillmentProvider = ManualFulfillmentService{}

// GetFulfillmentOptions mirrors getFulfillmentOptions.
func (ManualFulfillmentService) GetFulfillmentOptions(context.Context) ([]any, error) {
	a := dal.NewRecord()
	a.Set("id", "manual-fulfillment")
	b := dal.NewRecord()
	b.Set("id", "manual-fulfillment-return")
	b.Set("is_return", true)
	return []any{a, b}, nil
}

// ValidateFulfillmentData mirrors validateFulfillmentData: the data as is.
func (ManualFulfillmentService) ValidateFulfillmentData(_ context.Context, _, data, _ *dal.Record) (any, error) {
	return data, nil
}

// CalculatePrice mirrors calculatePrice.
func (ManualFulfillmentService) CalculatePrice(context.Context, *dal.Record, *dal.Record, *dal.Record) (*dal.Record, error) {
	return nil, errors.New("Manual fulfillment does not support price calculation")
}

// CanCalculate mirrors canCalculate.
func (ManualFulfillmentService) CanCalculate(context.Context, *dal.Record) (bool, error) {
	return false, nil
}

// ValidateOption mirrors validateOption.
func (ManualFulfillmentService) ValidateOption(context.Context, *dal.Record) (bool, error) {
	return true, nil
}

func emptyResult() *dal.Record {
	r := dal.NewRecord()
	r.Set("data", dal.NewRecord())
	r.Set("labels", []any{})
	return r
}

// CreateFulfillment mirrors createFulfillment: no data is sent anywhere.
func (ManualFulfillmentService) CreateFulfillment(context.Context, *dal.Record, []any, *dal.Record, *dal.Record) (*dal.Record, error) {
	return emptyResult(), nil
}

// CancelFulfillment mirrors cancelFulfillment.
func (ManualFulfillmentService) CancelFulfillment(context.Context, *dal.Record) (any, error) {
	return dal.NewRecord(), nil
}

// CreateReturnFulfillment mirrors createReturnFulfillment.
func (ManualFulfillmentService) CreateReturnFulfillment(context.Context, *dal.Record) (*dal.Record, error) {
	return emptyResult(), nil
}
