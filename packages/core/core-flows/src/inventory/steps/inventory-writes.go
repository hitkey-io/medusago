package steps

import (
	"context"
	"fmt"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func rget(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func rowsOf(res any) []*dal.Record {
	if r, ok := res.(*dal.Record); ok {
		return []*dal.Record{r}
	}
	l, _ := res.([]*dal.Record)
	return l
}

func withoutKeys(r *dal.Record, keys ...string) *dal.Record {
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

// UpdateInventoryItems mirrors updateInventoryItems (location_levels
// dropped).
func UpdateInventoryItems(ctx context.Context, svc commonsteps.ModuleService, input []*dal.Record) ([]*dal.Record, error) {
	in := make([]any, len(input))
	for i, r := range input {
		in[i] = withoutKeys(r, "location_levels")
	}
	res, err := svc.Internal("InventoryItem").Update(ctx, in)
	return rowsOf(res), err
}

// UpdateInventoryItemsStep mirrors updateInventoryItemsStep.
var UpdateInventoryItemsStep = composer.CreateStep("update-inventory-items-step",
	func(ctx composer.StepExecutionContext, input []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(input) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
		}
		svc := inventory(ctx.Container)
		f := dal.NewRecord()
		ids := make([]any, len(input))
		for i, r := range input {
			ids[i] = rget(r, "id")
		}
		f.Set("id", ids)
		before, err := svc.Internal("InventoryItem").List(ctx.Context, f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		out, err := UpdateInventoryItems(ctx.Context, svc, input)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(out, any(before)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		before, _ := data.([]*dal.Record)
		if len(before) == 0 {
			return nil
		}
		_, err := UpdateInventoryItems(ctx.Context, inventory(ctx.Container), before)
		return err
	})

// EnsureInventoryLevels mirrors ensureInventoryLevels (without the quantity
// validation).
func EnsureInventoryLevels(ctx context.Context, svc commonsteps.ModuleService, data []*dal.Record) ([]*dal.Record, error) {
	or := []any{}
	idF := dal.NewRecord()
	idF.Set("id", []any{})
	or = append(or, idF)
	for _, d := range data {
		f := dal.NewRecord()
		f.Set("location_id", rget(d, "location_id"))
		f.Set("inventory_item_id", rget(d, "inventory_item_id"))
		or = append(or, f)
	}
	filter := dal.NewRecord()
	filter.Set("$or", or)
	levels, err := svc.Internal("InventoryLevel").List(ctx, filter, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	has := map[string]bool{}
	for _, l := range levels {
		has[fmt.Sprint(rget(l, "inventory_item_id"))+"|"+fmt.Sprint(rget(l, "location_id"))] = true
	}
	var missing []string
	for _, d := range data {
		if !has[fmt.Sprint(rget(d, "inventory_item_id"))+"|"+fmt.Sprint(rget(d, "location_id"))] {
			missing = append(missing, fmt.Sprintf("Item %v is not stocked at location %v", rget(d, "inventory_item_id"), rget(d, "location_id")))
		}
	}
	if len(missing) > 0 {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, strings.Join(missing, ", "))
	}
	return levels, nil
}

// UpdateInventoryLevels mirrors updateInventoryLevels.
func UpdateInventoryLevels(ctx context.Context, svc commonsteps.ModuleService, updates []*dal.Record) ([]*dal.Record, error) {
	input := make([]*dal.Record, len(updates))
	for i, u := range updates {
		input[i] = withoutKeys(u, "reserved_quantity")
	}
	levels, err := EnsureInventoryLevels(ctx, svc, input)
	if err != nil {
		return nil, err
	}
	idOf := map[string]any{}
	for _, l := range levels {
		idOf[fmt.Sprint(rget(l, "inventory_item_id"))+"|"+fmt.Sprint(rget(l, "location_id"))] = rget(l, "id")
	}
	in := make([]any, len(input))
	for i, u := range input {
		n := dal.NewRecord()
		n.Set("id", idOf[fmt.Sprint(rget(u, "inventory_item_id"))+"|"+fmt.Sprint(rget(u, "location_id"))])
		for _, k := range u.Keys() {
			v, _ := u.Get(k)
			n.Set(k, v)
		}
		in[i] = n
	}
	res, err := svc.Internal("InventoryLevel").Update(ctx, in)
	return rowsOf(res), err
}

// UpdateInventoryLevelsStep mirrors updateInventoryLevelsStep.
var UpdateInventoryLevelsStep = composer.CreateStep("update-inventory-levels-step",
	func(ctx composer.StepExecutionContext, input []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(input) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
		}
		svc := inventory(ctx.Container)
		or := make([]any, len(input))
		for i, u := range input {
			f := dal.NewRecord()
			f.Set("inventory_item_id", rget(u, "inventory_item_id"))
			f.Set("location_id", rget(u, "location_id"))
			or[i] = f
		}
		filter := dal.NewRecord()
		filter.Set("$or", or)
		before, err := svc.Internal("InventoryLevel").List(ctx.Context, filter, utilsmodulessdk.FindConfig{})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		out, err := UpdateInventoryLevels(ctx.Context, svc, input)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(out, any(before)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		before, _ := data.([]*dal.Record)
		if len(before) == 0 {
			return nil
		}
		in := make([]*dal.Record, len(before))
		for i, b := range before {
			n := dal.NewRecord()
			for _, k := range []string{"inventory_item_id", "location_id", "stocked_quantity", "incoming_quantity", "metadata"} {
				if v, ok := b.Get(k); ok {
					n.Set(k, v)
				}
			}
			in[i] = n
		}
		_, err := UpdateInventoryLevels(ctx.Context, inventory(ctx.Container), in)
		return err
	})

// DeleteInventoryLevelsStep mirrors deleteEntitiesStep with
// softDeleteInventoryLevels / restoreInventoryLevels.
var DeleteInventoryLevelsStep = composer.CreateStep("delete-entities-step",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		if len(ids) == 0 {
			return composer.NewStepResponse[any](nil, any([]any{})), nil
		}
		if _, err := inventory(ctx.Container).Internal("InventoryLevel").SoftDelete(ctx.Context, ids); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := inventory(ctx.Container).Internal("InventoryLevel").Restore(ctx.Context, ids)
		return err
	})
