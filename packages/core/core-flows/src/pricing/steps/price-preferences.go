package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreatePricePreferencesStep mirrors createPricePreferencesStep.
var CreatePricePreferencesStep = composer.CreateStep("create-price-preferences",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := pricing(ctx.Container).CreatePricePreferences(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		ids := make([]any, len(created))
		for i, c := range created {
			ids[i], _ = c.Get("id")
		}
		return composer.NewStepResponse(created, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := pricing(ctx.Container).Internal("PricePreference").Delete(ctx.Context, ids)
		return err
	})

// UpdatePricePreferencesInput mirrors UpdatePricePreferencesWorkflowInput.
type UpdatePricePreferencesInput struct {
	Selector *dal.Record
	Update   *dal.Record
}

// UpdatePricePreferencesStep mirrors updatePricePreferencesStep.
var UpdatePricePreferencesStep = composer.CreateStep("update-price-preferences",
	func(ctx composer.StepExecutionContext, in UpdatePricePreferencesInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := pricing(ctx.Container)
		selects, relations := common.GetSelectsAndRelationsFromObjectArray([]common.ObjectLike{in.Update}, common.SelectsAndRelationsOptions{}, "")
		prev, err := svc.ListBySuffix(ctx.Context, "PricePreferences", in.Selector, utilsmodulessdk.FindConfig{Select: selects, Relations: relations})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		out, err := svc.UpdatePricePreferences(ctx.Context, in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(out, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		_, err := pricing(ctx.Container).UpsertPricePreferences(ctx.Context, prev)
		return err
	})

// DeletePricePreferencesStep mirrors deletePricePreferencesStep.
var DeletePricePreferencesStep = composer.CreateStep("delete-price-preferences",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		if _, err := pricing(ctx.Container).Internal("PricePreference").SoftDelete(ctx.Context, ids); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := pricing(ctx.Container).Internal("PricePreference").Restore(ctx.Context, ids)
		return err
	})

type pricePreferencesAsArrayCompensation struct {
	prevData   []*dal.Record
	newDataIDs []any
}

// UpdatePricePreferencesAsArrayStep mirrors updatePricePreferencesAsArrayStep.
var UpdatePricePreferencesAsArrayStep = composer.CreateStep("update-price-preferences-as-array",
	func(ctx composer.StepExecutionContext, input []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		svc := pricing(ctx.Container)
		if len(input) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any(pricePreferencesAsArrayCompensation{})), nil
		}
		or := make([]any, len(input))
		for i, entry := range input {
			a, _ := entry.Get("attribute")
			v, _ := entry.Get("value")
			if a == nil || a == "" || v == nil || v == "" {
				return composer.StepResponse[[]*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Attribute and value must be provided when updating price preferences")
			}
			f := dal.NewRecord()
			f.Set("attribute", a)
			f.Set("value", v)
			or[i] = f
		}
		filter := dal.NewRecord()
		filter.Set("$or", or)
		prev, err := svc.ListBySuffix(ctx.Context, "PricePreferences", filter, utilsmodulessdk.FindConfig{})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		toUpsert := make([]*dal.Record, len(input))
		for i, entry := range input {
			a, _ := entry.Get("attribute")
			v, _ := entry.Get("value")
			var match *dal.Record
			for _, p := range prev {
				pa, _ := p.Get("attribute")
				pv, _ := p.Get("value")
				if pa == a && pv == v {
					match = p
					break
				}
			}
			r := dal.NewRecord()
			if match != nil {
				id, _ := match.Get("id")
				r.Set("id", id)
			}
			r.Set("attribute", a)
			r.Set("value", v)
			if t, ok := entry.Get("is_tax_inclusive"); ok && t != nil {
				r.Set("is_tax_inclusive", t)
			} else if match != nil {
				t, _ := match.Get("is_tax_inclusive")
				r.Set("is_tax_inclusive", t)
			}
			toUpsert[i] = r
		}
		upserted, err := svc.UpsertPricePreferences(ctx.Context, toUpsert)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		prevIDs := map[any]bool{}
		for _, p := range prev {
			id, _ := p.Get("id")
			prevIDs[id] = true
		}
		var newIDs []any
		for _, u := range upserted {
			id, _ := u.Get("id")
			if !prevIDs[id] {
				newIDs = append(newIDs, id)
			}
		}
		return composer.NewStepResponse(upserted, any(pricePreferencesAsArrayCompensation{prevData: prev, newDataIDs: newIDs})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		c, ok := data.(pricePreferencesAsArrayCompensation)
		if !ok {
			return nil
		}
		svc := pricing(ctx.Container)
		if len(c.prevData) > 0 {
			if _, err := svc.UpsertPricePreferences(ctx.Context, c.prevData); err != nil {
				return err
			}
		}
		if len(c.newDataIDs) > 0 {
			_, err := svc.Internal("PricePreference").Delete(ctx.Context, c.newDataIDs)
			return err
		}
		return nil
	})
