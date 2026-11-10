package workflows

import (
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	pricingsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/pricing/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// VariantPrices is one entry of UpsertVariantPricesWorkflowInput.variantPrices.
type VariantPrices struct {
	VariantID any
	ProductID any
	// Prices nil = undefined.
	Prices []any
}

// UpsertVariantPricesWorkflowInput mirrors UpsertVariantPricesWorkflowInput.
type UpsertVariantPricesWorkflowInput struct {
	VariantPrices      []VariantPrices
	PreviousVariantIDs []any
}

// UpsertVariantPricesWorkflow mirrors upsertVariantPricesWorkflow.
var UpsertVariantPricesWorkflow = composer.CreateWorkflow("upsert-variant-prices",
	func(e *composer.Execution, in UpsertVariantPricesWorkflowInput) (any, error) {
		current := map[string]bool{}
		for _, v := range in.VariantPrices {
			current[fmt.Sprint(v.VariantID)] = true
		}
		removed := []any{}
		for _, id := range in.PreviousVariantIDs {
			if !current[fmt.Sprint(id)] {
				removed = append(removed, id)
			}
		}
		keys := dal.NewRecord()
		keys.Set("variant_id", removed)
		entry := dal.NewRecord()
		entry.Set(utilsmodulessdk.PRODUCT, keys)
		if _, err := commonsteps.RemoveRemoteLinkStep.Config("remove-variant-link-step", composer.StepOptions{}).Run(e, []*dal.Record{entry}); err != nil {
			return nil, err
		}

		previous := map[string]bool{}
		for _, id := range in.PreviousVariantIDs {
			previous[fmt.Sprint(id)] = true
		}
		var existing, created []VariantPrices
		for _, v := range in.VariantPrices {
			if previous[fmt.Sprint(v.VariantID)] {
				existing = append(existing, v)
			} else {
				created = append(created, v)
			}
		}
		existingIDs := []any{}
		for _, v := range existing {
			existingIDs = append(existingIDs, v.VariantID)
		}
		filters := dal.NewRecord()
		filters.Set("variant_id", existingIDs)
		vars := dal.NewRecord()
		vars.Set("filters", filters)
		links, err := commonsteps.UseRemoteQueryStep.Run(e, commonsteps.RemoteStepInput{EntryPoint: "product_variant_price_set", Fields: []string{"variant_id", "price_set_id"}, Variables: vars})
		if err != nil {
			return nil, err
		}
		priceSetByVariant := map[string]any{}
		for _, l := range links.Data {
			if r, ok := l.(*dal.Record); ok {
				vid, _ := r.Get("variant_id")
				psid, _ := r.Get("price_set_id")
				priceSetByVariant[fmt.Sprint(vid)] = psid
			}
		}
		priceSets := []*dal.Record{}
		for _, v := range existing {
			psid, ok := priceSetByVariant[fmt.Sprint(v.VariantID)]
			if !ok || psid == nil || v.Prices == nil {
				continue
			}
			ps := dal.NewRecord()
			ps.Set("id", psid)
			ps.Set("prices", v.Prices)
			priceSets = append(priceSets, ps)
		}
		if _, err := pricingsteps.UpdatePriceSetsStep.Run(e, pricingsteps.UpdatePriceSetsStepInput{Batch: true, PriceSets: priceSets}); err != nil {
			return nil, err
		}

		toCreate := make([]*dal.Record, len(created))
		for i, v := range created {
			ps := dal.NewRecord()
			if v.Prices != nil {
				ps.Set("prices", v.Prices)
			}
			toCreate[i] = ps
		}
		sets, err := pricingsteps.CreatePriceSetsStep.Run(e, toCreate)
		if err != nil {
			return nil, err
		}
		pricingLinks := make([]steps.VariantPriceSetLink, len(created))
		for i, v := range created {
			psid, _ := sets[i].Get("id")
			pricingLinks[i] = steps.VariantPriceSetLink{VariantID: v.VariantID, PriceSetID: psid}
		}
		if _, err := steps.CreateVariantPricingLinkStep.Run(e, pricingLinks); err != nil {
			return nil, err
		}
		events := make([]any, len(in.VariantPrices))
		for i, v := range in.VariantPrices {
			r := dal.NewRecord()
			r.Set("id", v.VariantID)
			events[i] = r
		}
		_, err = commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-variant.updated", Data: events})
		return nil, err
	})
