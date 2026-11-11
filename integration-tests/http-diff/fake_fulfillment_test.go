package httpdiff

import (
	"context"
	"errors"

	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
)

// testCalculatedResolve is the provider's resolve path in the reference
// application's medusa-config.js (src/modules/test-calculated-fulfillment).
const testCalculatedResolve = "./src/modules/test-calculated-fulfillment"

// testCalculatedFulfillment is the fulfillment provider of the reference
// application that calculates shipping prices: base (option data, default 3)
// + 1.5 per unit + extra (method data).
type testCalculatedFulfillment struct{}

func init() {
	modules.Registry[testCalculatedResolve] = &modulessdk.ModuleExports{
		ProviderIdentifier: "test-calculated",
		NewProvider: func(modulessdk.ProviderOptions) (any, error) {
			return testCalculatedFulfillment{}, nil
		},
	}
}

func (testCalculatedFulfillment) GetFulfillmentOptions(context.Context) ([]any, error) {
	a := dal.NewRecord()
	a.Set("id", "calc-standard")
	a.Set("name", "Calculated")
	b := dal.NewRecord()
	b.Set("id", "calc-return")
	b.Set("is_return", true)
	return []any{a, b}, nil
}

func (testCalculatedFulfillment) ValidateFulfillmentData(_ context.Context, _, data, context_ *dal.Record) (any, error) {
	out := dal.NewRecord()
	if data != nil {
		for _, k := range data.Keys() {
			v, _ := data.Get(k)
			out.Set(k, v)
		}
	}
	out.Set("validated", true)
	items, _ := rget(context_, "items").([]any)
	n := len(items)
	if n == 0 {
		if l, ok := rget(context_, "items").([]*dal.Record); ok {
			n = len(l)
		}
	}
	out.Set("items_count", float64(n))
	return out, nil
}

func (testCalculatedFulfillment) ValidateOption(context.Context, *dal.Record) (bool, error) {
	return true, nil
}

func (testCalculatedFulfillment) CanCalculate(context.Context, *dal.Record) (bool, error) {
	return true, nil
}

func number(v any, fallback float64) float64 {
	if v == nil {
		return fallback
	}
	f, _ := totals.Convert(v).Float64()
	return f
}

func (testCalculatedFulfillment) CalculatePrice(_ context.Context, optionData, data, context_ *dal.Record) (*dal.Record, error) {
	if fail := rget(data, "fail"); fail != nil && fail != false && fail != "" && fail != float64(0) {
		return nil, errors.New("Test provider failed to calculate the price")
	}
	quantity := 0.0
	switch items := rget(context_, "items").(type) {
	case []any:
		for _, it := range items {
			r, _ := it.(*dal.Record)
			quantity += number(rget(r, "quantity"), 0)
		}
	case []*dal.Record:
		for _, r := range items {
			quantity += number(rget(r, "quantity"), 0)
		}
	}
	base := number(rget(optionData, "base"), 3)
	extra := number(rget(data, "extra"), 0)
	out := dal.NewRecord()
	out.Set("calculated_amount", base+1.5*quantity+extra)
	ti := rget(optionData, "tax_inclusive")
	out.Set("is_calculated_price_tax_inclusive", ti != nil && ti != false && ti != "" && ti != float64(0))
	return out, nil
}

func (testCalculatedFulfillment) CreateFulfillment(context.Context, *dal.Record, []any, *dal.Record, *dal.Record) (*dal.Record, error) {
	out := dal.NewRecord()
	d := dal.NewRecord()
	d.Set("tracking_number", "TEST-TRACK")
	out.Set("data", d)
	out.Set("labels", []any{})
	return out, nil
}

func (testCalculatedFulfillment) CancelFulfillment(context.Context, *dal.Record) (any, error) {
	return dal.NewRecord(), nil
}

func (testCalculatedFulfillment) CreateReturnFulfillment(context.Context, *dal.Record) (*dal.Record, error) {
	out := dal.NewRecord()
	out.Set("data", dal.NewRecord())
	out.Set("labels", []any{})
	return out, nil
}
