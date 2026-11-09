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

// DeleteProductVariantsInput mirrors DeleteProductVariantsWorkflowInput.
type DeleteProductVariantsInput struct {
	IDs []any
}

// DeleteProductVariantsWorkflow mirrors deleteProductVariantsWorkflow.
var DeleteProductVariantsWorkflow = composer.CreateWorkflow("delete-product-variants",
	func(e *composer.Execution, input DeleteProductVariantsInput) (any, error) {
		filters := dal.NewRecord()
		filters.Set("id", input.IDs)
		res, err := commonsteps.UseQueryGraphStep.Run(e, commonsteps.UseQueryGraphStepInput{GraphInput: query.GraphInput{
			Entity: "variants", Fields: []string{"id", "manage_inventory", "inventory.id", "inventory.variants.id"}, Filters: filters,
		}})
		if err != nil {
			return nil, err
		}
		inner := dal.NewRecord()
		inner.Set("variant_id", input.IDs)
		inner.Set("product_variant_id", input.IDs)
		entry := dal.NewRecord()
		entry.Set(utilsmodulessdk.PRODUCT, inner)
		if _, err := commonsteps.RemoveRemoteLinkStep.Config("remove-variant-link-step", composer.StepOptions{}).Run(e, []*dal.Record{entry}); err != nil {
			return nil, err
		}
		variants := anyRecords(res.Data)
		inSet := map[string]bool{}
		for _, v := range variants {
			id, _ := v.Get("id")
			inSet[fmt.Sprint(id)] = true
		}
		toDelete := []any{}
		seen := map[string]bool{}
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
					if !inSet[fmt.Sprint(id)] {
						all = false
						break
					}
				}
				id, _ := item.Get("id")
				if all && !seen[fmt.Sprint(id)] {
					seen[fmt.Sprint(id)] = true
					toDelete = append(toDelete, id)
				}
			}
		}
		if _, err := inventoryworkflows.DeleteInventoryItemWorkflow.RunAsStep(e, toDelete); err != nil {
			return nil, err
		}
		if _, err := steps.DeleteProductVariantsStep.Run(e, input.IDs); err != nil {
			return nil, err
		}
		events := make([]any, len(input.IDs))
		for i, id := range input.IDs {
			ev := dal.NewRecord()
			ev.Set("id", id)
			events[i] = ev
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-variant.deleted", Data: events}); err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("ids", input.IDs)
		if err := composer.CreateHook(e, "productVariantsDeleted", hookInput); err != nil {
			return nil, err
		}
		return nil, nil
	})
