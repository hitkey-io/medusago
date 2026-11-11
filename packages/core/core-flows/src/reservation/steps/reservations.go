// Package steps mirrors @medusajs/core-flows/src/reservation/steps: every
// reservation change runs while its inventory items are locked
// (locking.execute).
package steps

import (
	"context"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	lockingsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/locking/steps"
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

func get(r *dal.Record, key string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(key)
	return v
}

func anyList(v any) []any {
	l, _ := v.([]any)
	return l
}

// inventoryItemIDs maps the rows to their inventory_item_id.
func inventoryItemIDs(rows []*dal.Record) []any {
	out := make([]any, len(rows))
	for i, r := range rows {
		out[i] = get(r, "inventory_item_id")
	}
	return out
}

// withLock mirrors locking.execute(Array.from(new Set(inventoryItemIds)), job).
func withLock(ctx composer.StepExecutionContext, itemIDs []any, fn func(ctx context.Context) error) error {
	return lockingsteps.Execute(ctx.Context, ctx.Container, lockingsteps.UniqueKeys(itemIDs), fn)
}

func compensation(pairs ...any) *dal.Record {
	r := dal.NewRecord()
	for i := 0; i+1 < len(pairs); i += 2 {
		r.Set(pairs[i].(string), pairs[i+1])
	}
	return r
}

// CreateReservationsStep mirrors createReservationsStep.
var CreateReservationsStep = composer.CreateStep("create-reservations-step",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		itemIDs := inventoryItemIDs(data)
		var created []*dal.Record
		err := withLock(ctx, itemIDs, func(lctx context.Context) error {
			var err error
			created, err = service(ctx.Container).CreateReservationItems(lctx, data)
			return err
		})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(compensation("reservations", commonsteps.RecordIDs(created), "inventoryItemIds", itemIDs))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		r, _ := data.(*dal.Record)
		ids := anyList(get(r, "reservations"))
		if len(ids) == 0 {
			return nil
		}
		return withLock(ctx, anyList(get(r, "inventoryItemIds")), func(lctx context.Context) error {
			return service(ctx.Container).DeleteReservationItems(lctx, ids)
		})
	})

type updateCompensation struct {
	before  []*dal.Record
	keys    [][]string
	itemIDs []any
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
		itemIDs := inventoryItemIDs(before)
		var updated []*dal.Record
		err = withLock(ctx, itemIDs, func(lctx context.Context) error {
			var err error
			updated, err = svc.UpdateReservationItems(lctx, data)
			return err
		})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(updated, any(updateCompensation{before: before, keys: keys, itemIDs: itemIDs})), nil
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
		return withLock(ctx, c.itemIDs, func(lctx context.Context) error {
			_, err := service(ctx.Container).UpdateReservationItems(lctx, restore)
			return err
		})
	})

// DeleteReservationsStep mirrors deleteReservationsStep.
var DeleteReservationsStep = composer.CreateStep("delete-reservations",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		f := dal.NewRecord()
		f.Set("id", ids)
		rows, err := service(ctx.Container).ListBySuffix(ctx.Context, "ReservationItems", f, utilsmodulessdk.FindConfig{Select: []string{"id", "inventory_item_id"}})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		itemIDs := inventoryItemIDs(rows)
		if err := withLock(ctx, itemIDs, func(lctx context.Context) error {
			return service(ctx.Container).SoftDeleteReservationItems(lctx, ids)
		}); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(compensation("ids", ids, "inventoryItemIds", itemIDs))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		r, _ := data.(*dal.Record)
		ids := anyList(get(r, "ids"))
		if len(ids) == 0 {
			return nil
		}
		return withLock(ctx, anyList(get(r, "inventoryItemIds")), func(lctx context.Context) error {
			return service(ctx.Container).RestoreReservationItems(lctx, ids)
		})
	})

type byLineItem interface {
	DeleteReservationItemsByLineItem(ctx context.Context, lineItemIDs []any) error
	RestoreReservationItemsByLineItem(ctx context.Context, lineItemIDs []any) error
}

// DeleteReservationsByLineItemsStep mirrors deleteReservationsByLineItemsStep.
var DeleteReservationsByLineItemsStep = composer.CreateStep("delete-reservations-by-line-items",
	func(ctx composer.StepExecutionContext, lineItemIDs []any) (composer.StepResponse[[]any], error) {
		f := dal.NewRecord()
		f.Set("line_item_id", lineItemIDs)
		rows, err := service(ctx.Container).ListBySuffix(ctx.Context, "ReservationItems", f, utilsmodulessdk.FindConfig{Select: []string{"id", "inventory_item_id"}})
		if err != nil {
			return composer.StepResponse[[]any]{}, err
		}
		itemIDs := inventoryItemIDs(rows)
		if err := withLock(ctx, itemIDs, func(lctx context.Context) error {
			return ctx.Container.Resolve(utilsmodulessdk.INVENTORY).(byLineItem).DeleteReservationItemsByLineItem(lctx, lineItemIDs)
		}); err != nil {
			return composer.StepResponse[[]any]{}, err
		}
		return composer.NewStepResponse(commonsteps.RecordIDs(rows), any(compensation("ids", lineItemIDs, "inventoryItemIds", itemIDs))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		r, _ := data.(*dal.Record)
		ids := anyList(get(r, "ids"))
		if len(ids) == 0 {
			return nil
		}
		return withLock(ctx, anyList(get(r, "inventoryItemIds")), func(lctx context.Context) error {
			return ctx.Container.Resolve(utilsmodulessdk.INVENTORY).(byLineItem).RestoreReservationItemsByLineItem(lctx, ids)
		})
	})
