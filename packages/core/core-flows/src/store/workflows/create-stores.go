package workflows

import (
	pricingsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/pricing/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/store/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateStoresWorkflowInput mirrors CreateStoresWorkflowInput.
type CreateStoresWorkflowInput struct {
	Stores []*dal.Record
}

func recs(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		out := make([]*dal.Record, 0, len(t))
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

// CreateStoresWorkflow mirrors createStoresWorkflow.
var CreateStoresWorkflow = composer.CreateWorkflow("create-stores",
	func(e *composer.Execution, input CreateStoresWorkflowInput) ([]*dal.Record, error) {
		normalized := make([]*dal.Record, len(input.Stores))
		for i, s := range input.Stores {
			n := dal.NewRecord()
			for _, k := range s.Keys() {
				v, _ := s.Get(k)
				n.Set(k, v)
			}
			if v, ok := s.Get("supported_currencies"); ok && v != nil {
				var currencies []any
				for _, c := range recs(v) {
					x := dal.NewRecord()
					code, _ := c.Get("currency_code")
					x.Set("currency_code", code)
					if d, ok := c.Get("is_default"); ok {
						x.Set("is_default", d)
					}
					currencies = append(currencies, x)
				}
				n.Set("supported_currencies", currencies)
			}
			normalized[i] = n
		}
		stores, err := steps.CreateStoresStep.Run(e, normalized)
		if err != nil {
			return nil, err
		}
		var prefs []*dal.Record
		index := map[any]int{}
		for _, s := range input.Stores {
			v, _ := s.Get("supported_currencies")
			for _, c := range recs(v) {
				code, _ := c.Get("currency_code")
				p := dal.NewRecord()
				p.Set("attribute", "currency_code")
				p.Set("value", code)
				if t, ok := c.Get("is_tax_inclusive"); ok {
					p.Set("is_tax_inclusive", t)
				}
				if i, seen := index[code]; seen {
					prefs[i] = p
					continue
				}
				index[code] = len(prefs)
				prefs = append(prefs, p)
			}
		}
		if prefs == nil {
			prefs = []*dal.Record{}
		}
		if _, err := pricingsteps.UpdatePricePreferencesAsArrayStep.Run(e, prefs); err != nil {
			return nil, err
		}
		return stores, nil
	})
