// Package workflows mirrors @medusajs/core-flows/src/stock-location/workflows.
package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	fulfillmentsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/steps"
	saleschannelsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/sales-channel/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/stock-location/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateStockLocationsWorkflow mirrors createStockLocationsWorkflow.
var CreateStockLocationsWorkflow = composer.CreateWorkflow("create-stock-locations-workflow",
	func(e *composer.Execution, locations []*dal.Record) ([]*dal.Record, error) {
		created, err := steps.CreateStockLocations.Run(e, locations)
		if err != nil {
			return nil, err
		}
		hook := dal.NewRecord()
		hook.Set("stockLocations", created)
		if err := composer.CreateHook(e, "stockLocationsCreated", hook); err != nil {
			return nil, err
		}
		return created, nil
	})

// DeleteStockLocationsWorkflow mirrors deleteStockLocationsWorkflow.
var DeleteStockLocationsWorkflow = composer.CreateWorkflow("delete-stock-locations-workflow",
	func(e *composer.Execution, ids []any) (any, error) {
		deleted, err := steps.DeleteStockLocationsStep.Run(e, ids)
		if err != nil {
			return nil, err
		}
		_, err = commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{deleted})
		return nil, err
	})

// UpdateStockLocationsInput mirrors UpdateStockLocationsWorkflowInput.
type UpdateStockLocationsInput struct {
	Selector *dal.Record
	Update   *dal.Record
}

// UpdateStockLocationsWorkflow mirrors updateStockLocationsWorkflow.
var UpdateStockLocationsWorkflow = composer.CreateWorkflow("update-stock-locations-workflow",
	func(e *composer.Execution, input UpdateStockLocationsInput) ([]*dal.Record, error) {
		q, err := commonsteps.UseQueryGraphStep.Config("get-stock-location", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "stock_location", Filters: input.Selector, Fields: []string{"id", "address.id"}},
		})
		if err != nil {
			return nil, err
		}
		var addresses []*dal.Record
		if a, ok := input.Update.Get("address"); ok && a != nil {
			address, _ := a.(*dal.Record)
			rows, _ := q.Data.([]any)
			for _, loc := range rows {
				locRec, _ := loc.(*dal.Record)
				var addrID any
				if locRec != nil {
					if ar, ok := locRec.Get("address"); ok {
						if arec, ok := ar.(*dal.Record); ok && arec != nil {
							addrID, _ = arec.Get("id")
						}
					}
				}
				n := dal.NewRecord()
				if addrID != nil && addrID != "" {
					n.Set("id", addrID)
				}
				for _, k := range address.Keys() {
					v, _ := address.Get(k)
					n.Set(k, v)
				}
				addresses = append(addresses, n)
			}
		}
		update := dal.NewRecord()
		for _, k := range input.Update.Keys() {
			if k == "address" || k == "address_id" {
				continue
			}
			v, _ := input.Update.Get(k)
			update.Set(k, v)
		}
		if addresses == nil {
			addresses = []*dal.Record{}
		}
		if _, err := steps.UpsertStockLocationAddressesStep.Run(e, addresses); err != nil {
			return nil, err
		}
		return steps.UpdateStockLocationsStep.Run(e, steps.UpdateStockLocationsStepInput{Selector: input.Selector, Update: update})
	})

// LinkSalesChannelsInput mirrors LinkWorkflowInput.
type LinkSalesChannelsInput struct {
	ID     string
	Add    []string
	Remove []string
}

// LinkSalesChannelsToStockLocationWorkflow mirrors linkSalesChannelsToStockLocationWorkflow.
var LinkSalesChannelsToStockLocationWorkflow = composer.CreateWorkflow("link-sales-channels-to-stock-location",
	func(e *composer.Execution, input LinkSalesChannelsInput) (any, error) {
		toAdd := make([]saleschannelsteps.LocationChannelLink, len(input.Add))
		for i, sc := range input.Add {
			toAdd[i] = saleschannelsteps.LocationChannelLink{SalesChannelID: sc, LocationID: input.ID}
		}
		toRemove := make([]saleschannelsteps.LocationChannelLink, len(input.Remove))
		for i, sc := range input.Remove {
			toRemove[i] = saleschannelsteps.LocationChannelLink{SalesChannelID: sc, LocationID: input.ID}
		}
		if _, err := saleschannelsteps.AssociateLocationsWithSalesChannelsStep.Run(e, toAdd); err != nil {
			return nil, err
		}
		_, err := saleschannelsteps.DetachLocationsFromSalesChannelsStep.Run(e, toRemove)
		return nil, err
	})

// CreateLocationFulfillmentSetInput mirrors CreateLocationFulfillmentSetWorkflowInputDTO.
type CreateLocationFulfillmentSetInput struct {
	LocationID string
	Name       any
	Type       any
}

// CreateLocationFulfillmentSetWorkflow mirrors createLocationFulfillmentSetWorkflow.
var CreateLocationFulfillmentSetWorkflow = composer.CreateWorkflow("create-location-fulfillment-set",
	func(e *composer.Execution, input CreateLocationFulfillmentSetInput) (any, error) {
		d := dal.NewRecord()
		d.Set("name", input.Name)
		d.Set("type", input.Type)
		created, err := fulfillmentsteps.CreateFulfillmentSets.Run(e, []*dal.Record{d})
		if err != nil {
			return nil, err
		}
		id, _ := created[0].Get("id")
		_, err = steps.AssociateFulfillmentSetsWithLocationStep.Run(e, []steps.LocationFulfillmentSets{{LocationID: input.LocationID, FulfillmentSetIDs: []any{id}}})
		return nil, err
	})
