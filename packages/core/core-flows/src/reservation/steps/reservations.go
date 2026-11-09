// Package steps mirrors @medusajs/core-flows/src/reservation/steps (the
// locking module is not ported: the steps run without locks).
package steps

import (
	"context"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type reservationService interface {
	CreateReservationItems(ctx context.Context, input []*dal.Record) ([]*dal.Record, error)
	UpdateReservationItems(ctx context.Context, input []*dal.Record) ([]*dal.Record, error)
	DeleteReservationItems(ctx context.Context, ids []any) error
	SoftDeleteReservationItems(ctx context.Context, ids []any) error
	RestoreReservationItems(ctx context.Context, ids []any) error
	ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error)
}

func service(c composer.Container) reservationService {
	return c.Resolve(utilsmodulessdk.INVENTORY).(reservationService)
}

// CreateReservationsStep mirrors createReservationsStep.
var CreateReservationsStep = composer.CreateStep("create-reservations-step",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := service(ctx.Container).CreateReservationItems(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(commonsteps.RecordIDs(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		return service(ctx.Container).DeleteReservationItems(ctx.Context, ids)
	})

type updateCompensation struct {
	before []*dal.Record
	keys   [][]string
}

// UpdateReservationsStep mirrors updateReservationsStep.
var UpdateReservationsStep = composer.CreateStep("update-reservations-step",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(data) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any(updateCompensation{})), nil
		}
		svc := service(ctx.Container)
		ids := make([]any, len(data))
		keys := make([][]string, len(data))
		for i, d := range data {
			ids[i], _ = d.Get("id")
			keys[i] = d.Keys()
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		before, err := svc.ListBySuffix(ctx.Context, "ReservationItems", f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		updated, err := svc.UpdateReservationItems(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(updated, any(updateCompensation{before: before, keys: keys})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		c, ok := data.(updateCompensation)
		if !ok || len(c.before) == 0 {
			return nil
		}
		fields := map[string]bool{"id": true}
		for _, ks := range c.keys {
			for _, k := range ks {
				fields[k] = true
			}
		}
		restore := make([]*dal.Record, len(c.before))
		for i, b := range c.before {
			r := dal.NewRecord()
			for _, k := range b.Keys() {
				if fields[k] {
					v, _ := b.Get(k)
					r.Set(k, v)
				}
			}
			restore[i] = r
		}
		_, err := service(ctx.Container).UpdateReservationItems(ctx.Context, restore)
		return err
	})

// DeleteReservationsStep mirrors deleteReservationsStep.
var DeleteReservationsStep = composer.CreateStep("delete-reservations",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		if err := service(ctx.Container).SoftDeleteReservationItems(ctx.Context, ids); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		return service(ctx.Container).RestoreReservationItems(ctx.Context, ids)
	})
