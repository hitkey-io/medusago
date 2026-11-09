// Package workflows mirrors @medusajs/core-flows/src/store/workflows.
package workflows

import (
	pricingsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/pricing/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/store/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdateStoresInput mirrors UpdateStoreWorkflowInput.
type UpdateStoresInput struct {
	Selector *dal.Record
	Update   *dal.Record
}

func currencies(update *dal.Record) []*dal.Record {
	v, _ := update.Get("supported_currencies")
	var out []*dal.Record
	switch t := v.(type) {
	case []any:
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
	case []*dal.Record:
		out = t
	}
	return out
}

// UpdateStoresWorkflow mirrors updateStoresWorkflow.
var UpdateStoresWorkflow = composer.CreateWorkflow("update-stores",
	func(e *composer.Execution, input UpdateStoresInput) ([]*dal.Record, error) {
		cs := currencies(input.Update)
		update := input.Update
		if len(cs) > 0 {
			update = dal.NewRecord()
			for _, k := range input.Update.Keys() {
				v, _ := input.Update.Get(k)
				update.Set(k, v)
			}
			normalized := make([]any, len(cs))
			for i, c := range cs {
				n := dal.NewRecord()
				code, _ := c.Get("currency_code")
				n.Set("currency_code", code)
				if d, ok := c.Get("is_default"); ok {
					n.Set("is_default", d)
				}
				normalized[i] = n
			}
			update.Set("supported_currencies", normalized)
		}
		stores, err := steps.UpdateStoresStep.Run(e, steps.UpdateStoresStepInput{Selector: input.Selector, Update: update})
		if err != nil {
			return nil, err
		}
		if len(cs) > 0 {
			prefs := make([]*dal.Record, len(cs))
			for i, c := range cs {
				p := dal.NewRecord()
				p.Set("attribute", "currency_code")
				code, _ := c.Get("currency_code")
				p.Set("value", code)
				if t, ok := c.Get("is_tax_inclusive"); ok {
					p.Set("is_tax_inclusive", t)
				}
				prefs[i] = p
			}
			if _, err := pricingsteps.UpdatePricePreferencesAsArrayStep.Run(e, prefs); err != nil {
				return nil, err
			}
		}
		return stores, nil
	})
