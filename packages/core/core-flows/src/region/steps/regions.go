// Package steps mirrors @medusajs/core-flows/src/region/steps.
package steps

import (
	"context"
	"fmt"
	"strings"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/modules/region/src/services"
)

func region(c composer.Container) *services.RegionModuleService {
	s, _ := c.Resolve(utilsmodulessdk.REGION).(*services.RegionModuleService)
	return s
}

// CreateRegionsStep mirrors createRegionsStep.
var CreateRegionsStep = composer.CreateStep("create-regions",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := region(ctx.Container).CreateRegions(ctx.Context, data)
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
		_, err := region(ctx.Container).Internal("Region").Delete(ctx.Context, ids)
		return err
	})

// UpdateRegionsStepInput mirrors { selector, update }.
type UpdateRegionsStepInput struct {
	Selector *dal.Record
	Update   *dal.Record
}

// UpdateRegionsStep mirrors updateRegionsStep.
var UpdateRegionsStep = composer.CreateStep("update-region",
	func(ctx composer.StepExecutionContext, in UpdateRegionsStepInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := region(ctx.Container)
		selects, relations := common.GetSelectsAndRelationsFromObjectArray([]common.ObjectLike{in.Update}, common.SelectsAndRelationsOptions{}, "")
		prev, err := svc.ListBySuffix(ctx.Context, "Regions", in.Selector, utilsmodulessdk.FindConfig{Select: selects, Relations: relations})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		if in.Update.Len() == 0 {
			return composer.NewStepResponse(prev, any([]*dal.Record{})), nil
		}
		out, err := svc.UpdateRegions(ctx.Context, in.Selector, in.Update)
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
		in := make([]*dal.Record, len(prev))
		for i, r := range prev {
			n := dal.NewRecord()
			for _, k := range []string{"id", "name", "currency_code", "metadata"} {
				v, _ := r.Get(k)
				n.Set(k, v)
			}
			if cs, ok := r.Get("countries"); ok {
				var isos []any
				if l, ok := cs.([]any); ok {
					for _, c := range l {
						if cr, ok := c.(*dal.Record); ok {
							iso, _ := cr.Get("iso_2")
							isos = append(isos, iso)
						}
					}
				}
				n.Set("countries", isos)
			}
			in[i] = n
		}
		_, err := region(ctx.Container).UpsertRegions(ctx.Context, in)
		return err
	})

// DeleteRegionsStep mirrors deleteRegionsStep.
var DeleteRegionsStep = composer.CreateStep("delete-regions",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		if err := region(ctx.Container).SoftDeleteRegions(ctx.Context, ids); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := region(ctx.Container).Internal("Region").Restore(ctx.Context, ids)
		return err
	})

// RegionPaymentProviders is { id, payment_providers? }.
type RegionPaymentProviders struct {
	ID               any
	PaymentProviders []any
	Defined          bool
}

type providerLinks struct {
	toCreate []*dal.Record
	toRemove []*dal.Record
}

func regionProviderLink(regionID, providerID any) *dal.Record {
	a := dal.NewRecord()
	a.Set("region_id", regionID)
	b := dal.NewRecord()
	b.Set("payment_provider_id", providerID)
	out := dal.NewRecord()
	out.Set(utilsmodulessdk.REGION, a)
	out.Set(utilsmodulessdk.PAYMENT, b)
	return out
}

// SetRegionsPaymentProvidersStep mirrors setRegionsPaymentProvidersStep.
var SetRegionsPaymentProvidersStep = composer.CreateStep("add-region-payment-providers-step",
	func(ctx composer.StepExecutionContext, input []RegionPaymentProviders) (composer.StepResponse[any], error) {
		var data []RegionPaymentProviders
		for _, d := range input {
			if d.Defined {
				data = append(data, d)
			}
		}
		if len(data) == 0 {
			return composer.NewStepResponse[any](nil), nil
		}
		var unique []any
		seen := map[string]bool{}
		for _, d := range data {
			for _, p := range d.PaymentProviders {
				if !seen[fmt.Sprint(p)] {
					seen[fmt.Sprint(p)] = true
					unique = append(unique, p)
				}
			}
		}
		payment, _ := ctx.Container.Resolve(utilsmodulessdk.PAYMENT).(suffixLister)
		in := dal.NewRecord()
		in.Set("$in", unique)
		f := dal.NewRecord()
		f.Set("id", in)
		f.Set("is_enabled", true)
		providers, err := payment.ListBySuffix(ctx.Context, "PaymentProviders", f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		found := map[string]bool{}
		for _, p := range providers {
			id, _ := p.Get("id")
			found[fmt.Sprint(id)] = true
		}
		var missing []string
		for _, p := range unique {
			if !found[fmt.Sprint(p)] {
				missing = append(missing, fmt.Sprint(p))
			}
		}
		if len(missing) > 0 {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Payment providers with ids %s not found or not enabled", strings.Join(missing, ", ")))
		}
		regionIDs := make([]any, len(data))
		for i, d := range data {
			regionIDs[i] = d.ID
		}
		lf := dal.NewRecord()
		lf.Set("region_id", regionIDs)
		link := commonsteps.LinkOf(ctx.Container)
		svc := link.GetLinkModule(utilsmodulessdk.REGION, "region_id", utilsmodulessdk.PAYMENT, "payment_provider_id")
		existing, err := svc.List(ctx.Context, lf, utilsmodulessdk.FindConfig{Select: []string{"region_id", "payment_provider_id"}})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		var toRemove, toCreate []*dal.Record
		for _, l := range existing {
			rid, _ := l.Get("region_id")
			pid, _ := l.Get("payment_provider_id")
			keep := false
			for _, d := range data {
				if fmt.Sprint(d.ID) == fmt.Sprint(rid) {
					for _, p := range d.PaymentProviders {
						if fmt.Sprint(p) == fmt.Sprint(pid) {
							keep = true
						}
					}
				}
			}
			if !keep {
				toRemove = append(toRemove, regionProviderLink(rid, pid))
			}
		}
		for _, d := range data {
			for _, p := range d.PaymentProviders {
				exists := false
				for _, l := range existing {
					rid, _ := l.Get("region_id")
					pid, _ := l.Get("payment_provider_id")
					if fmt.Sprint(rid) == fmt.Sprint(d.ID) && fmt.Sprint(pid) == fmt.Sprint(p) {
						exists = true
					}
				}
				if !exists {
					toCreate = append(toCreate, regionProviderLink(d.ID, p))
				}
			}
		}
		if len(toRemove) > 0 {
			if _, err := link.Dismiss(ctx.Context, toRemove); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		if len(toCreate) > 0 {
			if _, err := link.Create(ctx.Context, toCreate); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		return composer.NewStepResponse[any](nil, any(providerLinks{toCreate: toRemove, toRemove: toCreate})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		rb, ok := data.(providerLinks)
		if !ok {
			return nil
		}
		link := commonsteps.LinkOf(ctx.Container)
		if len(rb.toRemove) > 0 {
			if _, err := link.Dismiss(ctx.Context, rb.toRemove); err != nil {
				return err
			}
		}
		if len(rb.toCreate) > 0 {
			if _, err := link.Create(ctx.Context, rb.toCreate); err != nil {
				return err
			}
		}
		return nil
	})

type suffixLister interface {
	ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error)
}
