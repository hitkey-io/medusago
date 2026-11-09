package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/modules/product/src/services"
)

// UpdateBySelectorInput mirrors { selector, update } step inputs.
type UpdateBySelectorInput struct {
	Selector *dal.Record
	Update   *dal.Record
}

// prevDataFor mirrors the prevData lookup of the update steps: the
// selected rows with the fields and relations of the update.
func prevDataFor(ctx context.Context, s *services.ProductModuleService, suffix string, in UpdateBySelectorInput) ([]*dal.Record, error) {
	var objs []common.ObjectLike
	if in.Update != nil {
		objs = []common.ObjectLike{in.Update}
	}
	selects, relations := common.GetSelectsAndRelationsFromObjectArray(objs, common.SelectsAndRelationsOptions{}, "")
	return s.ListBySuffix(ctx, suffix, in.Selector, utilsmodulessdk.FindConfig{Select: selects, Relations: relations})
}

// softDeleteStep builds a delete step mirroring softDelete<X> / restore<X>.
func softDeleteStep(name, className string) *composer.Step[[]any, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
			if _, err := ProductService(ctx.Container).Internal(className).SoftDelete(ctx.Context, ids); err != nil {
				return composer.StepResponse[any]{}, err
			}
			return composer.NewStepResponse[any](nil, any(ids)), nil
		},
		func(ctx composer.StepExecutionContext, data any) error {
			ids, _ := data.([]any)
			if len(ids) == 0 {
				return nil
			}
			_, err := ProductService(ctx.Container).Internal(className).Restore(ctx.Context, ids)
			return err
		})
}

// createStep builds a create step whose compensation deletes the created
// rows.
func createStep(name, className string, create func(ctx context.Context, s *services.ProductModuleService, data []*dal.Record) ([]*dal.Record, error)) *composer.Step[[]*dal.Record, []*dal.Record] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
			created, err := create(ctx.Context, ProductService(ctx.Container), data)
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
			return composer.NewStepResponse(created, any(ids(created))), nil
		},
		func(ctx composer.StepExecutionContext, data any) error {
			idList, _ := data.([]any)
			if len(idList) == 0 {
				return nil
			}
			_, err := ProductService(ctx.Container).Internal(className).Delete(ctx.Context, idList)
			return err
		})
}

// updateStep builds an update step whose compensation upserts the previous
// data.
func updateStep(name, suffix string, update func(ctx context.Context, s *services.ProductModuleService, selector any, data *dal.Record) ([]*dal.Record, error), upsert func(ctx context.Context, s *services.ProductModuleService, data []*dal.Record) ([]*dal.Record, error)) *composer.Step[UpdateBySelectorInput, []*dal.Record] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, in UpdateBySelectorInput) (composer.StepResponse[[]*dal.Record], error) {
			svc := ProductService(ctx.Container)
			prev, err := prevDataFor(ctx.Context, svc, suffix, in)
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
			updated, err := update(ctx.Context, svc, in.Selector, in.Update)
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
			return composer.NewStepResponse(updated, any(prev)), nil
		},
		func(ctx composer.StepExecutionContext, data any) error {
			prev, _ := data.([]*dal.Record)
			if len(prev) == 0 {
				return nil
			}
			_, err := upsert(ctx.Context, ProductService(ctx.Container), prev)
			return err
		})
}
