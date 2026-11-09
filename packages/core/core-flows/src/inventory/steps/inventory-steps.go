// Package steps mirrors @medusajs/core-flows/src/inventory/steps.
package steps

import (
	"fmt"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func inventory(c composer.Container) commonsteps.ModuleService {
	return commonsteps.ModuleOf(c, utilsmodulessdk.INVENTORY)
}

func ids(rows []*dal.Record) []any {
	out := make([]any, len(rows))
	for i, r := range rows {
		out[i], _ = r.Get("id")
	}
	return out
}

func arrayDifference(a []any, b []any) []string {
	set := map[string]bool{}
	for _, x := range b {
		set[fmt.Sprint(x)] = true
	}
	var out []string
	for _, x := range a {
		if !set[fmt.Sprint(x)] {
			out = append(out, fmt.Sprint(x))
		}
	}
	return out
}

func remoteIDs(ctx composer.StepExecutionContext, entryPoint string, idsIn []any) ([]any, error) {
	q, _ := ctx.Container.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)
	vars := dal.NewRecord()
	vars.Set("id", idsIn)
	res, err := q.RemoteQuery(ctx.Context, query.RemoteQueryObjectFromString(query.RemoteQueryObjectConfig{EntryPoint: entryPoint, Variables: vars, Fields: []string{"id"}}))
	if err != nil {
		return nil, err
	}
	var out []any
	for _, row := range res.Data {
		if r, ok := row.(*dal.Record); ok {
			id, _ := r.Get("id")
			out = append(out, id)
		}
	}
	return out, nil
}

// ValidateInventoryItems mirrors validateInventoryItems.
var ValidateInventoryItems = composer.CreateStep("validate-inventory-items-step",
	func(ctx composer.StepExecutionContext, idsIn []any) (composer.StepResponse[any], error) {
		found, err := remoteIDs(ctx, "inventory_item", idsIn)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		if diff := arrayDifference(idsIn, found); len(diff) > 0 {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Inventory Items with ids: %s was not found", strings.Join(diff, ", ")))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// ValidateInventoryLocationsStep mirrors validateInventoryLocationsStep.
var ValidateInventoryLocationsStep = composer.CreateStep("validate-inventory-levels-step",
	func(ctx composer.StepExecutionContext, levels []*dal.Record) (composer.StepResponse[any], error) {
		var locs []any
		for _, l := range levels {
			v, _ := l.Get("location_id")
			locs = append(locs, v)
		}
		found, err := remoteIDs(ctx, "stock_location", locs)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		if diff := arrayDifference(locs, found); len(diff) > 0 {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Stock locations with ids: %s was not found", strings.Join(diff, ", ")))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// CreateInventoryItemsStep mirrors createInventoryItemsStep.
var CreateInventoryItemsStep = composer.CreateStep("create-inventory-items",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		toCreate := make([]*dal.Record, len(data))
		for i, d := range data {
			c := dal.NewRecord()
			for _, k := range d.Keys() {
				if k == "location_levels" {
					continue
				}
				v, _ := d.Get(k)
				c.Set(k, v)
			}
			toCreate[i] = c
		}
		items, err := inventory(ctx.Container).Internal("InventoryItem").Create(ctx.Context, toCreate)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(items, any(ids(items))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		idList, _ := data.([]any)
		if len(idList) == 0 {
			return nil
		}
		_, err := inventory(ctx.Container).Internal("InventoryItem").Delete(ctx.Context, idList)
		return err
	})

// CreateInventoryLevelsStep mirrors createInventoryLevelsStep.
var CreateInventoryLevelsStep = composer.CreateStep("create-inventory-levels",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		levels, err := inventory(ctx.Container).Internal("InventoryLevel").Create(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(levels, any(ids(levels))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		idList, _ := data.([]any)
		if len(idList) == 0 {
			return nil
		}
		_, err := inventory(ctx.Container).Internal("InventoryLevel").Delete(ctx.Context, idList)
		return err
	})

// ValidateInventoryDeleteStep mirrors validateInventoryDeleteStep.
var ValidateInventoryDeleteStep = composer.CreateStep("validate-inventory-item-delete",
	func(_ composer.StepExecutionContext, items []any) (composer.StepResponse[any], error) {
		var bad []string
		for _, it := range items {
			r, _ := it.(*dal.Record)
			if r == nil {
				continue
			}
			q, _ := r.Get("reserved_quantity")
			if f, ok := toFloat(q); ok && f > 0 {
				id, _ := r.Get("id")
				bad = append(bad, fmt.Sprint(id))
			}
		}
		if len(bad) > 0 {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA,
				fmt.Sprintf("Cannot remove following inventory item(s) since they have reservations: [%s].", strings.Join(bad, ", ")))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case string:
		var f float64
		_, err := fmt.Sscan(t, &f)
		return f, err == nil
	}
	return 0, false
}

// DeleteInventoryItemStep mirrors deleteInventoryItemStep.
var DeleteInventoryItemStep = composer.CreateStep("delete-inventory-item-step",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		if _, err := inventory(ctx.Container).Internal("InventoryItem").SoftDelete(ctx.Context, ids); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := inventory(ctx.Container).Internal("InventoryItem").Restore(ctx.Context, ids)
		return err
	})
