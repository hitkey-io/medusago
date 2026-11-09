package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdateProductsInput mirrors UpdateProductWorkflowInput in its selector
// form ({ selector, update }); Products (the batch form) is used when set.
type UpdateProductsInput struct {
	Selector       *dal.Record
	Update         *dal.Record
	Products       []*dal.Record
	Batch          bool
	AdditionalData *dal.Record
}

func linkDef(moduleA, keyA string, valA any, moduleB, keyB string, valB any) *dal.Record {
	a := dal.NewRecord()
	a.Set(keyA, valA)
	b := dal.NewRecord()
	b.Set(keyB, valB)
	out := dal.NewRecord()
	out.Set(moduleA, a)
	out.Set(moduleB, b)
	return out
}

// UpdateProductsWorkflow mirrors updateProductsWorkflow (variant prices and
// inventory handling only apply to nested variants, which the product
// module does not accept here yet).
var UpdateProductsWorkflow = composer.CreateWorkflow("update-products",
	func(e *composer.Execution, input UpdateProductsInput) ([]*dal.Record, error) {
		stepInput := steps.UpdateProductsStepInput{Batch: input.Batch}
		if input.Batch {
			for _, p := range input.Products {
				stepInput.Products = append(stepInput.Products, without(p, "sales_channels", "shipping_profile_id"))
			}
		} else {
			stepInput.Selector = input.Selector
			stepInput.Update = without(input.Update, "sales_channels", "shipping_profile_id")
		}
		updated, err := steps.UpdateProductsStep.Run(e, stepInput)
		if err != nil {
			return nil, err
		}

		var salesChannelLinks, shippingProfileLinks []*dal.Record
		var withSalesChannels, withShippingProfiles []any
		if input.Batch {
			for _, p := range input.Products {
				id, _ := p.Get("id")
				if scs, ok := p.Get("sales_channels"); ok && scs != nil {
					withSalesChannels = append(withSalesChannels, id)
					for _, sc := range anyRecords(scs) {
						scID, _ := sc.Get("id")
						salesChannelLinks = append(salesChannelLinks, linkDef(utilsmodulessdk.PRODUCT, "product_id", id, utilsmodulessdk.SALES_CHANNEL, "sales_channel_id", scID))
					}
				}
				if sp, ok := p.Get("shipping_profile_id"); ok && sp != nil {
					withShippingProfiles = append(withShippingProfiles, id)
					if s, isStr := sp.(string); isStr {
						shippingProfileLinks = append(shippingProfileLinks, linkDef(utilsmodulessdk.PRODUCT, "product_id", id, utilsmodulessdk.FULFILLMENT, "shipping_profile_id", s))
					}
				}
			}
		} else {
			scs, hasSC := input.Update.Get("sales_channels")
			sp, hasSP := input.Update.Get("shipping_profile_id")
			for _, p := range updated {
				id, _ := p.Get("id")
				if hasSC && scs != nil {
					withSalesChannels = append(withSalesChannels, id)
					for _, sc := range anyRecords(scs) {
						scID, _ := sc.Get("id")
						salesChannelLinks = append(salesChannelLinks, linkDef(utilsmodulessdk.PRODUCT, "product_id", id, utilsmodulessdk.SALES_CHANNEL, "sales_channel_id", scID))
					}
				}
				if hasSP && sp != nil {
					withShippingProfiles = append(withShippingProfiles, id)
					if s, isStr := sp.(string); isStr {
						shippingProfileLinks = append(shippingProfileLinks, linkDef(utilsmodulessdk.PRODUCT, "product_id", id, utilsmodulessdk.FULFILLMENT, "shipping_profile_id", s))
					}
				}
			}
		}
		if withSalesChannels == nil {
			withSalesChannels = []any{}
		}
		if withShippingProfiles == nil {
			withShippingProfiles = []any{}
		}

		currentLinks := func(name, entryPoint, key string, productIDs []any) ([]*dal.Record, error) {
			filters := dal.NewRecord()
			filters.Set("product_id", productIDs)
			vars := dal.NewRecord()
			vars.Set("filters", filters)
			res, err := commonsteps.UseRemoteQueryStep.Config(name, composer.StepOptions{}).Run(e, commonsteps.RemoteStepInput{EntryPoint: entryPoint, Fields: []string{"product_id", key}, Variables: vars})
			if err != nil {
				return nil, err
			}
			var out []*dal.Record
			for _, row := range anyRecords(res.Data) {
				pid, _ := row.Get("product_id")
				other, _ := row.Get(key)
				module := utilsmodulessdk.SALES_CHANNEL
				if key == "shipping_profile_id" {
					module = utilsmodulessdk.FULFILLMENT
				}
				out = append(out, linkDef(utilsmodulessdk.PRODUCT, "product_id", pid, module, key, other))
			}
			return out, nil
		}
		toDeleteSC, err := currentLinks("get-current-sales-channel-links-step", "product_sales_channel", "sales_channel_id", withSalesChannels)
		if err != nil {
			return nil, err
		}
		toDeleteSP, err := currentLinks("get-current-shipping-profile-links-step", "product_shipping_profile", "shipping_profile_id", withShippingProfiles)
		if err != nil {
			return nil, err
		}

		if _, err := commonsteps.DismissRemoteLinkStep.Config("delete-sales-channel-links-step", composer.StepOptions{}).Run(e, toDeleteSC); err != nil {
			return nil, err
		}
		if _, err := commonsteps.DismissRemoteLinkStep.Config("delete-shipping-profile-links-step", composer.StepOptions{}).Run(e, toDeleteSP); err != nil {
			return nil, err
		}

		err = composer.Parallelize(
			func() error {
				_, err := commonsteps.CreateRemoteLinkStep.Config("create-sales-channel-links-step", composer.StepOptions{}).Run(e, salesChannelLinks)
				return err
			},
			func() error {
				_, err := commonsteps.CreateRemoteLinkStep.Config("create-shipping-profile-links-step", composer.StepOptions{}).Run(e, shippingProfileLinks)
				return err
			},
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product.updated", Data: idEvents(updated)})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("products", updated)
		hookInput.Set("additional_data", input.AdditionalData)
		if err := composer.CreateHook(e, "productsUpdated", hookInput); err != nil {
			return nil, err
		}
		return updated, nil
	})
