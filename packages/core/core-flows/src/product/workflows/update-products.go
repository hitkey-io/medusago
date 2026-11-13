package workflows

import (
	"encoding/json"

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
		// the variant ids of the products whose variants are updated
		variantSelector := dal.NewRecord()
		if input.Batch {
			ids := []any{}
			for _, p := range input.Products {
				if v, ok := p.Get("variants"); ok && v != nil {
					id, _ := p.Get("id")
					ids = append(ids, id)
				}
			}
			f := dal.NewRecord()
			f.Set("id", ids)
			variantSelector.Set("filters", f)
		} else if v, ok := input.Update.Get("variants"); ok && v != nil {
			variantSelector.Set("filters", input.Selector)
		} else {
			f := dal.NewRecord()
			f.Set("id", []any{})
			variantSelector.Set("filters", f)
		}
		previous, err := commonsteps.UseRemoteQueryStep.Config("get-previous-products-variants-step", composer.StepOptions{}).Run(e, commonsteps.RemoteStepInput{EntryPoint: "product", Fields: []string{"variants.id"}, Variables: variantSelector})
		if err != nil {
			return nil, err
		}
		previousVariantIDs := []any{}
		for _, p := range anyRecords(previous.Data) {
			vs, _ := p.Get("variants")
			for _, v := range anyRecords(vs) {
				id, _ := v.Get("id")
				previousVariantIDs = append(previousVariantIDs, id)
			}
		}

		stripPrices := func(p *dal.Record) *dal.Record {
			out := without(p, "sales_channels", "shipping_profile_id")
			if vs, ok := out.Get("variants"); ok && vs != nil {
				var stripped []any
				for _, v := range anyRecords(vs) {
					stripped = append(stripped, without(v, "prices"))
				}
				if stripped == nil {
					stripped = []any{}
				}
				out.Set("variants", stripped)
			}
			return out
		}
		stepInput := steps.UpdateProductsStepInput{Batch: input.Batch}
		if input.Batch {
			for _, p := range input.Products {
				stepInput.Products = append(stepInput.Products, stripPrices(p))
			}
		} else {
			stepInput.Selector = input.Selector
			stepInput.Update = stripPrices(input.Update)
		}
		updated, err := steps.UpdateProductsStep.Run(e, stepInput)
		if err != nil {
			return nil, err
		}

		var inputVariants []*dal.Record
		if input.Batch {
			for _, p := range input.Products {
				vs, _ := p.Get("variants")
				inputVariants = append(inputVariants, anyRecords(vs)...)
			}
		} else if vs, ok := input.Update.Get("variants"); ok {
			inputVariants = anyRecords(vs)
		}
		toDismiss, toCreateInventory := []any{}, []any{}
		for _, v := range inputVariants {
			id, _ := v.Get("id")
			if id == nil || id == "" {
				continue
			}
			switch mi, _ := v.Get("manage_inventory"); mi {
			case false:
				toDismiss = append(toDismiss, id)
			case true:
				toCreateInventory = append(toCreateInventory, id)
			}
		}
		if err := composer.Parallelize(
			func() error {
				_, err := steps.DismissProductVariantsInventoryStep.Run(e, steps.VariantIDsInput{VariantIDs: toDismiss})
				return err
			},
			func() error {
				_, err := steps.CreateProductVariantsDefaultInventoryStep.Run(e, steps.VariantIDsInput{VariantIDs: toCreateInventory})
				return err
			},
		); err != nil {
			return nil, err
		}

		// prepareVariantPrices: relies on the order of the input and updated variants
		variantPrices := []VariantPrices{}
		pricesOf := func(v *dal.Record) []any {
			p, ok := v.Get("prices")
			if !ok || p == nil {
				return nil
			}
			l, _ := p.([]any)
			if l == nil {
				l = []any{}
			}
			return l
		}
		if input.Batch {
			for i, p := range input.Products {
				vs, _ := p.Get("variants")
				in := anyRecords(vs)
				if len(in) == 0 || i >= len(updated) {
					continue
				}
				uv, _ := updated[i].Get("variants")
				out := anyRecords(uv)
				pid, _ := updated[i].Get("id")
				for j, v := range in {
					var vid any
					if j < len(out) {
						vid, _ = out[j].Get("id")
					}
					variantPrices = append(variantPrices, VariantPrices{VariantID: vid, ProductID: pid, Prices: pricesOf(v)})
				}
			}
		} else if vs, ok := input.Update.Get("variants"); ok && len(anyRecords(vs)) > 0 {
			in := anyRecords(vs)
			for _, u := range updated {
				uv, _ := u.Get("variants")
				out := anyRecords(uv)
				pid, _ := u.Get("id")
				for j, v := range in {
					var vid any
					if j < len(out) {
						vid, _ = out[j].Get("id")
					}
					variantPrices = append(variantPrices, VariantPrices{VariantID: vid, ProductID: pid, Prices: pricesOf(v)})
				}
			}
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

		if _, err := UpsertVariantPricesWorkflow.RunAsStep(e, UpsertVariantPricesWorkflowInput{VariantPrices: variantPrices, PreviousVariantIDs: previousVariantIDs}); err != nil {
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
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product.updated", Data: idEvents(updated)})
				return err
			},
			func() error {
				_, err := commonsteps.CreateRemoteLinkStep.Config("create-sales-channel-links-step", composer.StepOptions{}).Run(e, salesChannelLinks)
				return err
			},
			func() error {
				_, err := commonsteps.CreateRemoteLinkStep.Config("create-shipping-profile-links-step", composer.StepOptions{}).Run(e, shippingProfileLinks)
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

// DecodeInput reads { selector, update } or { products } (the batch form)
// with additional_data.
func (in *UpdateProductsInput) DecodeInput(r *dal.Record) error {
	if v, ok := r.Get("products"); ok {
		in.Batch = true
		in.Products = anyRecords(v)
	} else {
		in.Selector, _ = get(r, "selector").(*dal.Record)
		in.Update, _ = get(r, "update").(*dal.Record)
		if in.Update == nil {
			in.Update = dal.NewRecord()
		}
	}
	in.AdditionalData, _ = get(r, "additional_data").(*dal.Record)
	return nil
}

// MarshalJSON writes the workflow input object.
func (in UpdateProductsInput) MarshalJSON() ([]byte, error) {
	out := dal.NewRecord()
	if in.Batch {
		out.Set("products", in.Products)
	} else {
		out.Set("selector", in.Selector)
		out.Set("update", in.Update)
	}
	if in.AdditionalData != nil {
		out.Set("additional_data", in.AdditionalData)
	}
	return json.Marshal(out)
}
