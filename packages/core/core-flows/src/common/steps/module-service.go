package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ModuleService is the generated module service surface (MedusaService)
// shared by modules without their own overrides.
type ModuleService interface {
	Internal(className string) *utilsmodulessdk.InternalService
	ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error)
}

// ModuleOf resolves a module service registered under key.
func ModuleOf(c composer.Container, key string) ModuleService {
	s, _ := c.Resolve(key).(ModuleService)
	return s
}

// UpdateBySelector mirrors the update<X>(selector, data) overrides that list
// the matching rows and update each { id, ...data }.
func UpdateBySelector(ctx context.Context, svc ModuleService, className string, selector, data *dal.Record) ([]*dal.Record, error) {
	rows, err := svc.Internal(className).List(ctx, selector, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []*dal.Record{}, nil
	}
	in := make([]any, len(rows))
	for i, r := range rows {
		n := dal.NewRecord()
		id, _ := r.Get("id")
		n.Set("id", id)
		for _, k := range data.Keys() {
			v, _ := data.Get(k)
			n.Set(k, v)
		}
		in[i] = n
	}
	return UpdateRecords(ctx, svc, className, in)
}

// UpdateRecords mirrors update<X>([{ id, ... }]).
func UpdateRecords(ctx context.Context, svc ModuleService, className string, in []any) ([]*dal.Record, error) {
	res, err := svc.Internal(className).Update(ctx, in)
	if err != nil {
		return nil, err
	}
	if r, ok := res.(*dal.Record); ok {
		return []*dal.Record{r}, nil
	}
	out, _ := res.([]*dal.Record)
	return out, nil
}

// PrevDataFor mirrors the prevData lookup of the update steps.
func PrevDataFor(ctx context.Context, svc ModuleService, suffix string, selector *dal.Record, updates ...*dal.Record) ([]*dal.Record, error) {
	objs := make([]common.ObjectLike, len(updates))
	for i, u := range updates {
		objs[i] = u
	}
	selects, relations := common.GetSelectsAndRelationsFromObjectArray(objs, common.SelectsAndRelationsOptions{}, "")
	return svc.ListBySuffix(ctx, suffix, selector, utilsmodulessdk.FindConfig{Select: selects, Relations: relations})
}

// RecordIDs returns the ids of records.
func RecordIDs(rs []*dal.Record) []any {
	out := make([]any, len(rs))
	for i, r := range rs {
		out[i], _ = r.Get("id")
	}
	return out
}

// CreateRecordsStep builds a create step whose compensation hard deletes the
// created rows.
func CreateRecordsStep(name, moduleKey, className string) *composer.Step[[]*dal.Record, []*dal.Record] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
			created, err := ModuleOf(ctx.Container, moduleKey).Internal(className).Create(ctx.Context, data)
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
			return composer.NewStepResponse(created, any(RecordIDs(created))), nil
		},
		func(ctx composer.StepExecutionContext, data any) error {
			ids, _ := data.([]any)
			if len(ids) == 0 {
				return nil
			}
			_, err := ModuleOf(ctx.Container, moduleKey).Internal(className).Delete(ctx.Context, ids)
			return err
		})
}

// SoftDeleteRecordsStep builds a soft delete step restoring on compensation.
func SoftDeleteRecordsStep(name, moduleKey, className string) *composer.Step[[]any, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
			if _, err := ModuleOf(ctx.Container, moduleKey).Internal(className).SoftDelete(ctx.Context, ids); err != nil {
				return composer.StepResponse[any]{}, err
			}
			return composer.NewStepResponse[any](nil, any(ids)), nil
		},
		func(ctx composer.StepExecutionContext, data any) error {
			ids, _ := data.([]any)
			if len(ids) == 0 {
				return nil
			}
			_, err := ModuleOf(ctx.Container, moduleKey).Internal(className).Restore(ctx.Context, ids)
			return err
		})
}

// Caller is a module service with the generated methods callable by name
// (MedusaService.Call).
type Caller interface {
	Call(ctx context.Context, method string, args []any) (any, error)
}

// SoftDeleteVia mirrors service.softDelete<X>(ids): the soft deleted
// entities' linkable keys (falls back to the internal service of
// className without them).
func SoftDeleteVia(ctx context.Context, svc interface {
	Internal(className string) *utilsmodulessdk.InternalService
}, method, className string, ids []any) (any, error) {
	if c, ok := svc.(Caller); ok {
		return c.Call(ctx, method, []any{ids})
	}
	_, err := svc.Internal(className).SoftDelete(ctx, ids)
	return dal.NewRecord(), err
}
