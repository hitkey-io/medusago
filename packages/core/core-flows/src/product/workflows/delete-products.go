package workflows

import (
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	inventoryworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/inventory/workflows"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// DeleteProductsInput mirrors DeleteProductsWorkflowInput.
type DeleteProductsInput struct {
	IDs []any
}

func anyRecords(v any) []*dal.Record {
	l, _ := v.([]any)
	out := make([]*dal.Record, 0, len(l))
	for _, x := range l {
		if r, ok := x.(*dal.Record); ok && r != nil {
			out = append(out, r)
		}
	}
	return out
}

// DeleteProductsWorkflow mirrors deleteProductsWorkflow.
var DeleteProductsWorkflow = composer.CreateWorkflow("delete-products",
	func(e *composer.Execution, input DeleteProductsInput) (any, error) {
		filters := dal.NewRecord()
		filters.Set("id", input.IDs)
		res, err := commonsteps.UseQueryGraphStep.Config("query-products-with-options-step", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{GraphInput: query.GraphInput{
			Entity: "product",
			Fields: []string{"id", "variants.id", "variants.manage_inventory", "variants.inventory.id", "variants.inventory.variants.id", "options.id", "options.is_exclusive"},
			Filters: filters,
		}})
		if err != nil {
			return nil, err
		}
		products := anyRecords(res.Data)

		var exclusive []any
		seen := map[string]bool{}
		var variants []*dal.Record
		var allVariantIDs []any
		for _, p := range products {
			opts, _ := p.Get("options")
			for _, o := range anyRecords(opts) {
				if ex, _ := o.Get("is_exclusive"); ex == true {
					id, _ := o.Get("id")
					if !seen[fmt.Sprint(id)] {
						seen[fmt.Sprint(id)] = true
						exclusive = append(exclusive, id)
					}
				}
			}
			vs, _ := p.Get("variants")
			for _, v := range anyRecords(vs) {
				variants = append(variants, v)
				id, _ := v.Get("id")
				allVariantIDs = append(allVariantIDs, id)
			}
		}
		variantSet := map[string]bool{}
		for _, id := range allVariantIDs {
			variantSet[fmt.Sprint(id)] = true
		}
		var toDeleteInventory []any
		invSeen := map[string]bool{}
		for _, v := range variants {
			if mi, _ := v.Get("manage_inventory"); mi != true {
				continue
			}
			inv, _ := v.Get("inventory")
			for _, item := range anyRecords(inv) {
				ivs, _ := item.Get("variants")
				all := true
				for _, iv := range anyRecords(ivs) {
					id, _ := iv.Get("id")
					if !variantSet[fmt.Sprint(id)] {
						all = false
						break
					}
				}
				if all {
					id, _ := item.Get("id")
					if !invSeen[fmt.Sprint(id)] {
						invSeen[fmt.Sprint(id)] = true
						toDeleteInventory = append(toDeleteInventory, id)
					}
				}
			}
		}
		if toDeleteInventory == nil {
			toDeleteInventory = []any{}
		}
		if _, err := inventoryworkflows.DeleteInventoryItemWorkflow.RunAsStep(e, toDeleteInventory); err != nil {
			return nil, err
		}

		if allVariantIDs == nil {
			allVariantIDs = []any{}
		}
		err = composer.Parallelize(
			func() error {
				inner := dal.NewRecord()
				inner.Set("variant_id", allVariantIDs)
				inner.Set("product_variant_id", allVariantIDs)
				inner.Set("product_id", input.IDs)
				entry := dal.NewRecord()
				entry.Set(utilsmodulessdk.PRODUCT, inner)
				_, err := commonsteps.RemoveRemoteLinkStep.Config("remove-product-variant-link-step", composer.StepOptions{}).Run(e, []*dal.Record{entry})
				return err
			},
			func() error {
				_, err := steps.DeleteProductsStep.Run(e, input.IDs)
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		if exclusive == nil {
			exclusive = []any{}
		}
		if _, err := DeleteProductOptionsWorkflow.RunAsStep(e, DeleteProductOptionsInput{IDs: exclusive}); err != nil {
			return nil, err
		}
		events := make([]any, len(input.IDs))
		for i, id := range input.IDs {
			ev := dal.NewRecord()
			ev.Set("id", id)
			events[i] = ev
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product.deleted", Data: events}); err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("ids", input.IDs)
		if err := composer.CreateHook(e, "productsDeleted", hookInput); err != nil {
			return nil, err
		}
		return nil, nil
	})
