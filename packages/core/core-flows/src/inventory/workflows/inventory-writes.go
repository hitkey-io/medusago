package workflows

import (
	"fmt"
	"strings"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/inventory/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdateInventoryItemsWorkflow mirrors updateInventoryItemsWorkflow.
var UpdateInventoryItemsWorkflow = composer.CreateWorkflow("update-inventory-items-workflow",
	func(e *composer.Execution, updates []*dal.Record) ([]*dal.Record, error) {
		updated, err := steps.UpdateInventoryItemsStep.Run(e, updates)
		if err != nil {
			return nil, err
		}
		_, err = commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "inventory-item.updated", Data: idEvents(updated)})
		return updated, err
	})

// UpdateInventoryLevelsWorkflow mirrors updateInventoryLevelsWorkflow.
var UpdateInventoryLevelsWorkflow = composer.CreateWorkflow("update-inventory-levels-workflow",
	func(e *composer.Execution, updates []*dal.Record) ([]*dal.Record, error) {
		updated, err := steps.UpdateInventoryLevelsStep.Run(e, updates)
		if err != nil {
			return nil, err
		}
		_, err = commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "inventory-level.updated", Data: idEvents(updated)})
		return updated, err
	})

func num(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case string:
		var f float64
		fmt.Sscan(t, &f)
		return f
	}
	return 0
}

func dedupe(list []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range list {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

type validateLevelsInput struct {
	Levels []*dal.Record
	Force  bool
}

// validateInventoryLevelsDelete mirrors validateInventoryLevelsDelete.
var validateInventoryLevelsDelete = composer.CreateStep("validate-inventory-levels-delete",
	func(ctx composer.StepExecutionContext, in validateLevelsInput) (composer.StepResponse[any], error) {
		var reserved, stocked []string
		for _, l := range in.Levels {
			r, _ := l.Get("reserved_quantity")
			inc, _ := l.Get("incoming_quantity")
			if num(r) > 0 || num(inc) > 0 {
				loc, _ := l.Get("location_id")
				reserved = append(reserved, fmt.Sprint(loc))
			}
		}
		if len(reserved) > 0 {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, fmt.Sprintf("Cannot remove Inventory Levels for %s because there are reserved or incoming items at the locations", strings.Join(dedupe(reserved), ", ")))
		}
		for _, l := range in.Levels {
			s, _ := l.Get("stocked_quantity")
			if !in.Force && num(s) > 0 {
				loc, _ := l.Get("location_id")
				stocked = append(stocked, fmt.Sprint(loc))
			}
		}
		if len(stocked) > 0 {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, fmt.Sprintf("Cannot remove Inventory Levels for %s because there are stocked items at the locations. Use force flag to delete anyway.", strings.Join(dedupe(stocked), ", ")))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// DeleteInventoryLevelsInput mirrors DeleteInventoryLevelsWorkflowInput:
// the inventory level filters with an optional force flag, in one object.
type DeleteInventoryLevelsInput struct {
	Filters *dal.Record
	Force   bool
}

// MarshalJSON serializes the input like upstream: the filters spread, then
// force when set.
func (in DeleteInventoryLevelsInput) MarshalJSON() ([]byte, error) {
	r := dal.NewRecord()
	if in.Filters != nil {
		for _, k := range in.Filters.Keys() {
			v, _ := in.Filters.Get(k)
			r.Set(k, v)
		}
	}
	if in.Force {
		r.Set("force", true)
	}
	return r.MarshalJSON()
}

// DecodeInput reads the upstream shape ({ ...filters, force }).
func (in *DeleteInventoryLevelsInput) DecodeInput(r *dal.Record) error {
	in.Filters = dal.NewRecord()
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		if k == "force" {
			in.Force, _ = v.(bool)
			continue
		}
		in.Filters.Set(k, v)
	}
	return nil
}

// DeleteInventoryLevelsWorkflow mirrors deleteInventoryLevelsWorkflow.
var DeleteInventoryLevelsWorkflow = composer.CreateWorkflow("delete-inventory-levels-workflow",
	func(e *composer.Execution, input DeleteInventoryLevelsInput) (any, error) {
		vars := dal.NewRecord()
		vars.Set("filters", input.Filters)
		res, err := commonsteps.UseRemoteQueryStep.Run(e, commonsteps.RemoteStepInput{EntryPoint: "inventory_levels", Fields: []string{"id", "stocked_quantity", "reserved_quantity", "location_id"}, Variables: vars})
		if err != nil {
			return nil, err
		}
		var levels []*dal.Record
		for _, r := range res.Data {
			if rec, ok := r.(*dal.Record); ok {
				levels = append(levels, rec)
			}
		}
		if _, err := validateInventoryLevelsDelete.Run(e, validateLevelsInput{Levels: levels, Force: input.Force}); err != nil {
			return nil, err
		}
		ids := make([]any, len(levels))
		for i, l := range levels {
			ids[i], _ = l.Get("id")
		}
		if _, err := steps.DeleteInventoryLevelsStep.Run(e, ids); err != nil {
			return nil, err
		}
		events := make([]any, len(ids))
		for i, id := range ids {
			ev := dal.NewRecord()
			ev.Set("id", id)
			events[i] = ev
		}
		_, err = commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "inventory-level.deleted", Data: events})
		return nil, err
	})

// BatchInventoryItemLevelsInput mirrors BatchInventoryItemLevelsWorkflowInput.
type BatchInventoryItemLevelsInput struct {
	Create []*dal.Record
	Update []*dal.Record
	Delete []any
	Force  bool
}

// BatchInventoryItemLevelsResult mirrors the workflow output.
type BatchInventoryItemLevelsResult struct {
	Created []*dal.Record
	Updated []*dal.Record
	Deleted []any
}

// BatchInventoryItemLevelsWorkflow mirrors batchInventoryItemLevelsWorkflow.
var BatchInventoryItemLevelsWorkflow = composer.CreateWorkflow("batch-inventory-item-levels-workflow",
	func(e *composer.Execution, input BatchInventoryItemLevelsInput) (BatchInventoryItemLevelsResult, error) {
		var out BatchInventoryItemLevelsResult
		create := input.Create
		if create == nil {
			create = []*dal.Record{}
		}
		update := input.Update
		if update == nil {
			update = []*dal.Record{}
		}
		del := input.Delete
		if del == nil {
			del = []any{}
		}
		err := composer.Parallelize(
			func() error {
				var err error
				out.Created, err = steps.CreateInventoryLevelsStep.Run(e, create)
				return err
			},
			func() error {
				var err error
				out.Updated, err = steps.UpdateInventoryLevelsStep.Run(e, update)
				return err
			},
			func() error {
				f := dal.NewRecord()
				f.Set("id", del)
				_, err := DeleteInventoryLevelsWorkflow.RunAsStep(e, DeleteInventoryLevelsInput{Filters: f, Force: input.Force})
				return err
			},
		)
		if err != nil {
			return out, err
		}
		err = composer.Parallelize(
			func() error {
				_, err := commonsteps.EmitEventStep.Config("emit-inventory-level-created", composer.StepOptions{}).Run(e, commonsteps.EmitEventStepInput{EventName: "inventory-level.created", Data: idEvents(out.Created)})
				return err
			},
			func() error {
				_, err := commonsteps.EmitEventStep.Config("emit-inventory-level-updated", composer.StepOptions{}).Run(e, commonsteps.EmitEventStepInput{EventName: "inventory-level.updated", Data: idEvents(out.Updated)})
				return err
			},
		)
		out.Deleted = input.Delete
		return out, err
	})
