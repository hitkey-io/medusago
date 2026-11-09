// Package steps mirrors @medusajs/core-flows/src/stock-location/steps.
package steps

import (
	"context"
	"fmt"
	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func svc(c composer.Container) commonsteps.ModuleService {
	return commonsteps.ModuleOf(c, utilsmodulessdk.STOCK_LOCATION)
}

type caller interface {
	Call(ctx context.Context, method string, args []any) (any, error)
}

// CreateStockLocations mirrors createStockLocations (step).
var CreateStockLocations = commonsteps.CreateRecordsStep("create-stock-locations", utilsmodulessdk.STOCK_LOCATION, "StockLocation")

// DeleteStockLocationsStep mirrors deleteStockLocationsStep: the soft delete
// returns the linkable keys used to remove the links.
var DeleteStockLocationsStep = composer.CreateStep("delete-stock-locations-step",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[*dal.Record], error) {
		c, ok := ctx.Container.Resolve(utilsmodulessdk.STOCK_LOCATION).(caller)
		if !ok {
			return composer.StepResponse[*dal.Record]{}, fmt.Errorf("stock location module does not support calls")
		}
		res, err := c.Call(ctx.Context, "softDeleteStockLocations", []any{ids})
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		out := dal.NewRecord()
		out.Set(utilsmodulessdk.STOCK_LOCATION, res)
		return composer.NewStepResponse(out, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := svc(ctx.Container).Internal("StockLocation").Restore(ctx.Context, ids)
		return err
	})

// UpdateStockLocationsStepInput mirrors the step input.
type UpdateStockLocationsStepInput struct {
	Selector *dal.Record
	Update   *dal.Record
}

// UpdateStockLocationsStep mirrors updateStockLocationsStep
// (stockLocationService_.update({ data, selector })).
var UpdateStockLocationsStep = composer.CreateStep("update-stock-locations-step",
	func(ctx composer.StepExecutionContext, in UpdateStockLocationsStepInput) (composer.StepResponse[[]*dal.Record], error) {
		s := svc(ctx.Container)
		prev, err := commonsteps.PrevDataFor(ctx.Context, s, "StockLocations", in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		u := dal.NewRecord()
		u.Set("data", in.Update)
		u.Set("selector", in.Selector)
		res, err := s.Internal("StockLocation").Update(ctx.Context, u)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		var out []*dal.Record
		if r, ok := res.(*dal.Record); ok {
			out = []*dal.Record{r}
		} else {
			out, _ = res.([]*dal.Record)
		}
		return composer.NewStepResponse(out, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		in := make([]*dal.Record, len(prev))
		for i, item := range prev {
			n := dal.NewRecord()
			id, _ := item.Get("id")
			name, _ := item.Get("name")
			n.Set("id", id)
			n.Set("name", name)
			if m, _ := item.Get("metadata"); m != nil {
				n.Set("metadata", m)
			}
			in[i] = n
		}
		_, err := svc(ctx.Container).Internal("StockLocation").Upsert(ctx.Context, in)
		return err
	})

type addressCompensation struct {
	toUpdate []*dal.Record
	toDelete []any
}

// UpsertStockLocationAddressesStep mirrors upsertStockLocationAddressesStep.
var UpsertStockLocationAddressesStep = composer.CreateStep("upsert-stock-location-addresses-step",
	func(ctx composer.StepExecutionContext, input []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		s := svc(ctx.Container)
		var ids []any
		for _, a := range input {
			if id, _ := a.Get("id"); id != nil && id != "" {
				ids = append(ids, id)
			}
		}
		f := dal.NewRecord()
		if ids == nil {
			ids = []any{}
		}
		f.Set("id", ids)
		prev, err := commonsteps.PrevDataFor(ctx.Context, s, "StockLocationAddresses", f, input...)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		var toCreate []*dal.Record
		var toUpdate []any
		for _, a := range input {
			if id, _ := a.Get("id"); id != nil && id != "" {
				toUpdate = append(toUpdate, a)
			} else {
				toCreate = append(toCreate, a)
			}
		}
		var out []*dal.Record
		if len(toCreate) > 0 {
			c, err := s.Internal("StockLocationAddress").Create(ctx.Context, toCreate)
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
			out = append(out, c...)
		}
		if len(toUpdate) > 0 {
			u, err := commonsteps.UpdateRecords(ctx.Context, s, "StockLocationAddress", toUpdate)
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
			out = append(out, u...)
		}
		prevIDs := map[any]bool{}
		for _, p := range prev {
			id, _ := p.Get("id")
			prevIDs[id] = true
		}
		var del []any
		for _, a := range out {
			if id, _ := a.Get("id"); !prevIDs[id] {
				del = append(del, id)
			}
		}
		return composer.NewStepResponse(out, any(addressCompensation{toUpdate: prev, toDelete: del})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		c, ok := data.(addressCompensation)
		if !ok {
			return nil
		}
		s := svc(ctx.Container)
		if len(c.toDelete) > 0 {
			if _, err := s.Internal("StockLocationAddress").Delete(ctx.Context, c.toDelete); err != nil {
				return err
			}
		}
		if len(c.toUpdate) > 0 {
			_, err := s.Internal("StockLocationAddress").Upsert(ctx.Context, c.toUpdate)
			return err
		}
		return nil
	})

// LocationFulfillmentSets is { location_id, fulfillment_set_ids }.
type LocationFulfillmentSets struct {
	LocationID        string
	FulfillmentSetIDs []any
}

// AssociateFulfillmentSetsWithLocationStep mirrors associateFulfillmentSetsWithLocationStep.
var AssociateFulfillmentSetsWithLocationStep = composer.CreateStep("associate-fulfillment-sets-with-location-step",
	func(ctx composer.StepExecutionContext, input []LocationFulfillmentSets) (composer.StepResponse[[]*dal.Record], error) {
		if len(input) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
		}
		var links []*dal.Record
		for _, l := range input {
			for _, id := range l.FulfillmentSetIDs {
				links = append(links, modulessdk.NewLinkDefinition("stock_location", "stock_location_id", l.LocationID, "fulfillment", "fulfillment_set_id", id))
			}
		}
		created, err := commonsteps.LinkOf(ctx.Container).Create(ctx.Context, links)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(links)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		links, _ := data.([]*dal.Record)
		if len(links) == 0 {
			return nil
		}
		_, err := commonsteps.LinkOf(ctx.Container).Dismiss(ctx.Context, links)
		return err
	})
