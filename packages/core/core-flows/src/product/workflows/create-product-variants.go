// Package workflows mirrors @medusajs/core-flows/src/product/workflows.
package workflows

import (
	"strings"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	commonworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/workflows"
	inventorysteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/inventory/steps"
	inventoryworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/inventory/workflows"
	pricingsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/pricing/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/utils"
	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func records(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		var out []*dal.Record
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
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

// CreateProductVariantsInput mirrors CreateProductVariantsWorkflowInput.
type CreateProductVariantsInput struct {
	ProductVariants []*dal.Record
	AdditionalData  *dal.Record
}

// CreateProductVariantsWorkflow mirrors createProductVariantsWorkflow.
var CreateProductVariantsWorkflow = composer.CreateWorkflow("create-product-variants",
	func(e *composer.Execution, input CreateProductVariantsInput) ([]*dal.Record, error) {
		variantsWithoutPrices := make([]*dal.Record, len(input.ProductVariants))
		for i, v := range input.ProductVariants {
			variantsWithoutPrices[i] = without(v, "prices")
		}
		created, err := steps.CreateProductVariantsStep.Run(e, variantsWithoutPrices)
		if err != nil {
			return nil, err
		}
		var inventoryItemIDs []any
		for _, v := range input.ProductVariants {
			for _, it := range records(get(v, "inventory_items")) {
				inventoryItemIDs = append(inventoryItemIDs, get(it, "inventory_item_id"))
			}
		}
		if _, err := inventorysteps.ValidateInventoryItems.Run(e, inventoryItemIDs); err != nil {
			return nil, err
		}

		// buildVariantItemCreateMap: 1-based variant index -> item input
		var itemIndexes []int
		var itemsToCreate []*dal.Record
		for i, variant := range created {
			in := input.ProductVariants[i]
			if mi, _ := get(variant, "manage_inventory").(bool); !mi || len(records(get(in, "inventory_items"))) > 0 {
				continue
			}
			itemIndexes = append(itemIndexes, i+1)
			itemsToCreate = append(itemsToCreate, utils.CreateDefaultInventoryItem(in))
		}
		createdItems, err := inventoryworkflows.CreateInventoryItemsWorkflow.RunAsStep(e, itemsToCreate)
		if err != nil {
			return nil, err
		}
		inventoryIndexMap := map[int]*dal.Record{}
		for j, idx := range itemIndexes {
			if j < len(createdItems) {
				inventoryIndexMap[idx] = createdItems[j]
			}
		}

		// buildLinksToCreate
		var erroredVariantIDs []string
		for i, variant := range created {
			seen := map[any]bool{}
			for _, it := range records(get(input.ProductVariants[i], "inventory_items")) {
				id := get(it, "inventory_item_id")
				if seen[id] {
					if vid, _ := get(variant, "id").(string); vid != "" {
						erroredVariantIDs = append(erroredVariantIDs, vid)
					}
					break
				}
				seen[id] = true
			}
		}
		if len(erroredVariantIDs) > 0 {
			return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Cannot associate duplicate inventory items to variant(s) "+strings.Join(erroredVariantIDs, "\n"))
		}
		buildLink := func(variantID, itemID, qty any) *dal.Record {
			l := modulessdk.NewLinkDefinition("product", "variant_id", variantID, "inventory", "inventory_item_id", itemID)
			d := dal.NewRecord()
			d.Set("required_quantity", qty)
			l.Set("data", d)
			return l
		}
		var links []*dal.Record
		for i, variant := range created {
			in := input.ProductVariants[i]
			if mi, _ := get(variant, "manage_inventory").(bool); !mi {
				continue
			}
			items := records(get(in, "inventory_items"))
			if len(items) == 0 {
				item := inventoryIndexMap[i+1]
				links = append(links, buildLink(get(variant, "id"), get(item, "id"), float64(1)))
				continue
			}
			for _, it := range items {
				qty := get(it, "required_quantity")
				if qty == nil {
					qty = float64(1)
				}
				links = append(links, buildLink(get(variant, "id"), get(it, "inventory_item_id"), qty))
			}
		}
		if _, err := commonworkflows.CreateLinksWorkflow.RunAsStep(e, links); err != nil {
			return nil, err
		}

		pricesToCreate := make([]*dal.Record, len(created))
		for i := range created {
			r := dal.NewRecord()
			if i < len(input.ProductVariants) {
				if p, ok := input.ProductVariants[i].Get("prices"); ok {
					r.Set("prices", p)
				}
			}
			pricesToCreate[i] = r
		}
		priceSets, err := pricingsteps.CreatePriceSetsStep.Run(e, pricesToCreate)
		if err != nil {
			return nil, err
		}
		pricingLinks := make([]steps.VariantPriceSetLink, len(created))
		for i, v := range created {
			pricingLinks[i] = steps.VariantPriceSetLink{VariantID: get(v, "id"), PriceSetID: get(priceSets[i], "id")}
		}
		if _, err := steps.CreateVariantPricingLinkStep.Run(e, pricingLinks); err != nil {
			return nil, err
		}
		response := make([]*dal.Record, len(created))
		for i, v := range created {
			r := without(v)
			prices := get(priceSets[i], "prices")
			if prices == nil {
				prices = []any{}
			}
			r.Set("prices", prices)
			response[i] = r
		}
		events := make([]any, len(response))
		for i, v := range response {
			ev := dal.NewRecord()
			ev.Set("id", get(v, "id"))
			events[i] = ev
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-variant.created", Data: events}); err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("product_variants", response)
		hookInput.Set("additional_data", input.AdditionalData)
		if err := CreateProductVariantsWorkflowHooks(e, hookInput); err != nil {
			return nil, err
		}
		return response, nil
	})

// CreateProductVariantsWorkflowHooks runs the productVariantsCreated hook.
var CreateProductVariantsWorkflowHooks = func(e *composer.Execution, input *dal.Record) error { return nil }

func init() {
	CreateProductVariantsWorkflowHooks = func(e *composer.Execution, input *dal.Record) error {
		return CreateProductVariantsWorkflow.CreateHook(e, "productVariantsCreated", input)
	}
}
