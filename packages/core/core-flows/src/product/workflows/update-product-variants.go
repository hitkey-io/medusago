package workflows

import (
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	pricingsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/pricing/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdateProductVariantsInput mirrors UpdateProductVariantsWorkflowInput:
// ProductVariants (when Batch) or Selector/Update.
type UpdateProductVariantsInput struct {
	Batch           bool
	ProductVariants []*dal.Record
	Selector        *dal.Record
	Update          *dal.Record
	AdditionalData  *dal.Record
}

// UpdateProductVariantsWorkflow mirrors updateProductVariantsWorkflow.
var UpdateProductVariantsWorkflow = composer.CreateWorkflow("update-product-variants",
	func(e *composer.Execution, input UpdateProductVariantsInput) ([]*dal.Record, error) {
		stepIn := steps.UpdateProductVariantsStepInput{Batch: input.Batch}
		if input.Batch {
			for _, v := range input.ProductVariants {
				stepIn.ProductVariants = append(stepIn.ProductVariants, without(v, "prices"))
			}
		} else {
			stepIn.Selector = input.Selector
			stepIn.Update = without(input.Update, "prices")
		}
		updated, err := steps.UpdateProductVariantsStep.Run(e, stepIn)
		if err != nil {
			return nil, err
		}

		var dismiss, create []any
		if input.Batch {
			for _, v := range input.ProductVariants {
				id, _ := v.Get("id")
				if id == nil || id == "" {
					continue
				}
				switch mi, _ := v.Get("manage_inventory"); mi {
				case false:
					dismiss = append(dismiss, id)
				case true:
					create = append(create, id)
				}
			}
		} else if input.Update != nil {
			mi, _ := input.Update.Get("manage_inventory")
			for _, v := range updated {
				id, _ := v.Get("id")
				switch mi {
				case false:
					dismiss = append(dismiss, id)
				case true:
					create = append(create, id)
				}
			}
		}
		err = composer.Parallelize(
			func() error {
				_, err := steps.DismissProductVariantsInventoryStep.Run(e, steps.VariantIDsInput{VariantIDs: dismiss})
				return err
			},
			func() error {
				_, err := steps.CreateProductVariantsDefaultInventoryStep.Run(e, steps.VariantIDsInput{VariantIDs: create})
				return err
			},
		)
		if err != nil {
			return nil, err
		}

		var variantIDs []any
		if input.Batch {
			withPrices := map[string]bool{}
			for _, v := range input.ProductVariants {
				if p, ok := v.Get("prices"); ok && p != nil {
					id, _ := v.Get("id")
					withPrices[fmt.Sprint(id)] = true
				}
			}
			for _, v := range updated {
				id, _ := v.Get("id")
				if withPrices[fmt.Sprint(id)] {
					variantIDs = append(variantIDs, id)
				}
			}
		} else if p, ok := input.Update.Get("prices"); ok && p != nil {
			for _, v := range updated {
				id, _ := v.Get("id")
				variantIDs = append(variantIDs, id)
			}
		}
		links, err := steps.GetVariantPricingLinkStep.Run(e, variantIDs)
		if err != nil {
			return nil, err
		}
		priceIn := pricingsteps.UpdatePriceSetsStepInput{}
		if len(links) > 0 {
			if input.Batch {
				priceIn.Batch = true
				for _, l := range links {
					for _, v := range input.ProductVariants {
						if id, _ := v.Get("id"); fmt.Sprint(id) == fmt.Sprint(l.VariantID) {
							ps := dal.NewRecord()
							ps.Set("id", l.PriceSetID)
							prices, _ := v.Get("prices")
							ps.Set("prices", prices)
							priceIn.PriceSets = append(priceIn.PriceSets, ps)
							break
						}
					}
				}
			} else {
				setIDs := make([]any, len(links))
				for i, l := range links {
					setIDs[i] = l.PriceSetID
				}
				priceIn.Selector = dal.NewRecord()
				priceIn.Selector.Set("id", setIDs)
				priceIn.Update = dal.NewRecord()
				prices, _ := input.Update.Get("prices")
				priceIn.Update.Set("prices", prices)
			}
		}
		updatedSets, err := pricingsteps.UpdatePriceSetsStep.Run(e, priceIn)
		if err != nil {
			return nil, err
		}

		response := make([]*dal.Record, len(updated))
		for i, v := range updated {
			r := dal.NewRecord()
			for _, k := range v.Keys() {
				x, _ := v.Get(k)
				r.Set(k, x)
			}
			id, _ := v.Get("id")
			var set any
			for _, l := range links {
				if fmt.Sprint(l.VariantID) == fmt.Sprint(id) {
					for _, s := range updatedSets {
						if sid, _ := s.Get("id"); fmt.Sprint(sid) == fmt.Sprint(l.PriceSetID) {
							set = s
						}
					}
				}
			}
			if set != nil {
				r.Set("price_set", set)
			} else {
				// price_set: undefined is not serialized
				r.Delete("price_set")
			}
			response[i] = r
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-variant.updated", Data: idEvents(response)}); err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("product_variants", response)
		hookInput.Set("additional_data", input.AdditionalData)
		if err := composer.CreateHook(e, "productVariantsUpdated", hookInput); err != nil {
			return nil, err
		}
		return response, nil
	})
