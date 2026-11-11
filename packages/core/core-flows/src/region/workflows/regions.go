// Package workflows mirrors @medusajs/core-flows/src/region/workflows.
package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	pricingsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/pricing/steps"
	pricingworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/pricing/workflows"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/region/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func without(r *dal.Record, keys ...string) *dal.Record {
	out := dal.NewRecord()
	skip := map[string]bool{}
	for _, k := range keys {
		skip[k] = true
	}
	for _, k := range r.Keys() {
		if !skip[k] {
			v, _ := r.Get(k)
			out.Set(k, v)
		}
	}
	return out
}

// providers returns the payment_providers list and whether it is defined
// (isDefined: present and not undefined; null counts as defined upstream,
// but the validators never produce null).
func providers(r *dal.Record) ([]any, bool) {
	v, ok := r.Get("payment_providers")
	if !ok {
		return nil, false
	}
	switch t := v.(type) {
	case []any:
		return t, true
	case []string:
		out := make([]any, len(t))
		for i, s := range t {
			out[i] = s
		}
		return out, true
	case nil:
		return nil, true
	}
	return nil, true
}

func idEvents(records []*dal.Record) []any {
	out := make([]any, len(records))
	for i, r := range records {
		ev := dal.NewRecord()
		ev.Set("id", get(r, "id"))
		out[i] = ev
	}
	return out
}

// CreateRegionsInput mirrors CreateRegionsWorkflowInput.
type CreateRegionsInput struct {
	Regions []*dal.Record
}

// CreateRegionsWorkflow mirrors createRegionsWorkflow.
var CreateRegionsWorkflow = composer.CreateWorkflow("create-regions",
	func(e *composer.Execution, input CreateRegionsInput) ([]*dal.Record, error) {
		data := make([]*dal.Record, len(input.Regions))
		for i, r := range input.Regions {
			data[i] = without(r, "is_tax_inclusive", "payment_providers")
		}
		regions, err := steps.CreateRegionsStep.Run(e, data)
		if err != nil {
			return nil, err
		}
		providerData := make([]steps.RegionPaymentProviders, len(input.Regions))
		prefs := make([]*dal.Record, len(input.Regions))
		for i, r := range input.Regions {
			p, defined := providers(r)
			providerData[i] = steps.RegionPaymentProviders{ID: get(regions[i], "id"), PaymentProviders: p, Defined: defined}
			pref := dal.NewRecord()
			pref.Set("attribute", "region_id")
			pref.Set("value", get(regions[i], "id"))
			if v, ok := r.Get("is_tax_inclusive"); ok {
				pref.Set("is_tax_inclusive", v)
			}
			prefs[i] = pref
		}
		err = composer.Parallelize(
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "region.created", Data: idEvents(regions)})
				return err
			},
			// the price preferences (one insert) complete before the payment
			// provider links (a lookup, then the links) upstream
			func() error {
				_, err := pricingworkflows.CreatePricePreferencesWorkflow.RunAsStep(e, prefs)
				return err
			},
			func() error {
				_, err := steps.SetRegionsPaymentProvidersStep.Run(e, providerData)
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		return regions, nil
	})

// UpdateRegionsInput mirrors UpdateRegionsWorkflowInput.
type UpdateRegionsInput struct {
	Selector *dal.Record
	Update   *dal.Record
}

// UpdateRegionsWorkflow mirrors updateRegionsWorkflow.
var UpdateRegionsWorkflow = composer.CreateWorkflow("update-regions",
	func(e *composer.Execution, input UpdateRegionsInput) ([]*dal.Record, error) {
		p, defined := providers(input.Update)
		isTaxInclusive, hasTaxInclusive := input.Update.Get("is_tax_inclusive")
		regions, err := steps.UpdateRegionsStep.Run(e, steps.UpdateRegionsStepInput{
			Selector: input.Selector,
			Update:   without(input.Update, "payment_providers", "is_tax_inclusive"),
		})
		if err != nil {
			return nil, err
		}
		providerData := make([]steps.RegionPaymentProviders, len(regions))
		for i, r := range regions {
			providerData[i] = steps.RegionPaymentProviders{ID: get(r, "id"), PaymentProviders: p, Defined: defined}
		}
		if hasTaxInclusive {
			or := make([]any, len(regions))
			for i, r := range regions {
				s := dal.NewRecord()
				s.Set("attribute", "region_id")
				s.Set("value", get(r, "id"))
				or[i] = s
			}
			selector := dal.NewRecord()
			selector.Set("$or", or)
			update := dal.NewRecord()
			update.Set("is_tax_inclusive", isTaxInclusive)
			if _, err := pricingworkflows.UpdatePricePreferencesWorkflow.RunAsStep(e, pricingsteps.UpdatePricePreferencesInput{Selector: selector, Update: update}); err != nil {
				return nil, err
			}
		}
		err = composer.Parallelize(
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "region.updated", Data: idEvents(regions)})
				return err
			},
			func() error {
				_, err := steps.SetRegionsPaymentProvidersStep.Run(e, providerData)
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		return regions, nil
	})

// DeleteRegionsInput mirrors DeleteRegionsWorkflowInput.
type DeleteRegionsInput struct {
	IDs []any
}

// DeleteRegionsWorkflow mirrors deleteRegionsWorkflow.
var DeleteRegionsWorkflow = composer.CreateWorkflow("delete-regions",
	func(e *composer.Execution, input DeleteRegionsInput) (any, error) {
		if _, err := steps.DeleteRegionsStep.Run(e, input.IDs); err != nil {
			return nil, err
		}
		events := make([]any, len(input.IDs))
		for i, id := range input.IDs {
			ev := dal.NewRecord()
			ev.Set("id", id)
			events[i] = ev
		}
		keys := dal.NewRecord()
		keys.Set("region_id", input.IDs)
		entry := dal.NewRecord()
		entry.Set(utilsmodulessdk.REGION, keys)
		if _, err := commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{entry}); err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "region.deleted", Data: events}); err != nil {
			return nil, err
		}
		return nil, nil
	})
