// Package workflows mirrors @medusajs/core-flows/src/cart/workflows.
package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// VariantsAndItemsInput mirrors GetVariantsAndItemsWithPricesWorkflowInput.
type VariantsAndItemsInput struct {
	Cart                    *dal.Record
	Items                   []*dal.Record
	SetPricingContextResult *dal.Record
	VariantIDs              []any
	VariantFields           []string
}

// VariantsAndItems mirrors GetVariantsAndItemsWithPricesWorkflowOutput.
type VariantsAndItems struct {
	Variants  []*dal.Record `json:"variants"`
	LineItems []*dal.Record `json:"lineItems"` // { selector: { id }, data }
}

func itemIDLike(item *dal.Record) string {
	if id := Str(item, "id"); id != "" {
		return id
	}
	return SimpleHash(JSONString(item))
}

func itemsOf(in VariantsAndItemsInput) []*dal.Record {
	if in.Items != nil {
		return in.Items
	}
	return Records(Get(in.Cart, "items"))
}

type prepareInput struct {
	cart      *dal.Record
	items     []*dal.Record
	variants  []*dal.Record
	priceSets map[string]*dal.Record
}

var prepareVariantsAndItemsWithPricesStep = composer.CreateStep("prepare-variants-and-items-with-prices",
	func(ctx composer.StepExecutionContext, in prepareInput) (composer.StepResponse[VariantsAndItems], error) {
		var priceNotFound, notFoundOrPublished []string
		byID := map[string]*dal.Record{}
		for _, v := range in.variants {
			if _, ok := byID[Str(v, "id")]; !ok {
				byID[Str(v, "id")] = v
			}
		}
		lineItems := []*dal.Record{}
		for _, item := range in.items {
			idLike := itemIDLike(item)
			ps := in.priceSets[idLike]
			if ps == nil {
				ps = in.priceSets[Str(item, "variant_id")]
			}
			isCustom := Truthy(Get(item, "is_custom_price"))
			if v, ok := item.Get("is_custom_price"); !ok || v == nil {
				isCustom = Has(item, "unit_price")
			}
			variantID := Str(item, "variant_id")
			if ps == nil && variantID != "" && !isCustom {
				priceNotFound = append(priceNotFound, variantID)
			}
			variant := byID[variantID]
			if (variantID != "" && variant == nil) || (variant != nil && Get(Rec(variant, "product"), "status") != "published") {
				notFoundOrPublished = append(notFoundOrPublished, variantID)
			}
			if variant != nil {
				if ps != nil {
					variant.Set("calculated_price", ps)
				} else {
					variant.Set("calculated_price", nil)
				}
			}
			inclusive := Get(item, "is_tax_inclusive")
			if inclusive == nil {
				inclusive = Get(ps, "is_calculated_price_tax_inclusive")
			}
			pin := PrepareLineItemDataInput{Item: item, Variant: variant, CartID: Str(in.cart, "id"), UnitPrice: Get(item, "unit_price"), IsTaxInclusive: Truthy(inclusive), IsCustomPrice: isCustom}
			if variant != nil && !isCustom && ps != nil {
				pin.UnitPrice = Get(ps, "calculated_amount")
				pin.IsTaxInclusive = Truthy(Get(ps, "is_calculated_price_tax_inclusive"))
			}
			prepared, err := PrepareLineItemData(pin)
			if err != nil {
				return composer.StepResponse[VariantsAndItems]{}, err
			}
			sel := dal.NewRecord()
			if id, ok := item.Get("id"); ok {
				sel.Set("id", id)
			}
			li := dal.NewRecord()
			li.Set("selector", sel)
			li.Set("data", prepared)
			lineItems = append(lineItems, li)
		}
		if len(notFoundOrPublished) > 0 {
			return composer.StepResponse[VariantsAndItems]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Variants "+strings.Join(notFoundOrPublished, ", ")+" do not exist or belong to a product that is not published")
		}
		if len(priceNotFound) > 0 {
			return composer.StepResponse[VariantsAndItems]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Variants with IDs "+strings.Join(priceNotFound, ", ")+" do not have a price")
		}
		return composer.NewStepResponse(VariantsAndItems{Variants: in.variants, LineItems: lineItems}), nil
	}, nil)

// GetVariantsAndItemsWithPrices mirrors getVariantsAndItemsWithPrices.
var GetVariantsAndItemsWithPrices = composer.CreateWorkflow("get-variant-items-with-prices-workflow",
	func(e *composer.Execution, in VariantsAndItemsInput) (VariantsAndItems, error) {
		variantIDs := in.VariantIDs
		if variantIDs == nil {
			seen := map[string]bool{}
			for _, i := range itemsOf(in) {
				if v := Str(i, "variant_id"); v != "" && !seen[v] {
					seen[v] = true
					variantIDs = append(variantIDs, v)
				}
			}
		}
		cart := in.Cart
		base := FilterObjectByKeys(cart, FieldsForPricingContext)
		set := func(k string, v any) {
			if v != nil {
				base.Set(k, v)
			} else {
				base.Delete(k)
			}
		}
		set("customer", Get(cart, "customer"))
		set("region", Get(cart, "region"))
		if in.SetPricingContextResult != nil {
			for _, k := range in.SetPricingContextResult.Keys() {
				v, _ := in.SetPricingContextResult.Get(k)
				base.Set(k, v)
			}
		}
		set("currency_code", Coalesce(Get(cart, "currency_code"), Get(Rec(cart, "region"), "currency_code")))
		set("region_id", Get(cart, "region_id"))
		set("customer_id", Get(cart, "customer_id"))
		var priceItems []steps.PriceItem
		for _, item := range itemsOf(in) {
			if Str(item, "variant_id") == "" {
				continue
			}
			c := Copy(base)
			c.Set("quantity", Get(item, "quantity"))
			c.Set("is_custom_price", Truthy(Get(item, "unit_price")))
			priceItems = append(priceItems, steps.PriceItem{ID: itemIDLike(item), VariantID: Str(item, "variant_id"), Context: c})
		}
		fields := in.VariantFields
		if fields == nil {
			fields = ProductVariantsFields
		}
		f := dal.NewRecord()
		if variantIDs == nil {
			variantIDs = []any{}
		}
		f.Set("id", variantIDs)
		res, err := commonsteps.UseQueryGraphStep.Config("fetch-variants", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{GraphInput: query.GraphInput{Entity: "variants", Fields: fields, Filters: f}, Options: joiner.Options{Cache: caching.WithTags(true, ProductVariantsCacheTags...)}})
		if err != nil {
			return VariantsAndItems{}, err
		}
		priceSets, err := steps.GetVariantPriceSetsStep.Run(e, priceItems)
		if err != nil {
			return VariantsAndItems{}, err
		}
		variants := Records(res.Data)
		if variants == nil {
			variants = []*dal.Record{}
		}
		return prepareVariantsAndItemsWithPricesStep.Run(e, prepareInput{cart: cart, items: itemsOf(in), variants: variants, priceSets: priceSets})
	})

// ConfirmVariantInventoryOutput mirrors ConfirmVariantInventoryWorkflowOutput
// ({ items }).
type ConfirmVariantInventoryOutput struct {
	Items []ConfirmInventoryItem `json:"items"`
}

// ConfirmVariantInventoryWorkflow mirrors confirmVariantInventoryWorkflow.
var ConfirmVariantInventoryWorkflow = composer.CreateWorkflow("confirm-item-inventory",
	func(e *composer.Execution, in ConfirmVariantInventoryInput) (*ConfirmVariantInventoryOutput, error) {
		items, err := PrepareConfirmInventoryInput(in)
		if err != nil {
			return nil, err
		}
		if _, err := steps.ConfirmInventoryStep.Run(e, items); err != nil {
			return nil, err
		}
		return &ConfirmVariantInventoryOutput{Items: items}, nil
	})
