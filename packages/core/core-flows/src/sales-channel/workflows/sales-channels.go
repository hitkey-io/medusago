// Package workflows mirrors @medusajs/core-flows/src/sales-channel/workflows.
package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/sales-channel/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func idEvents(list []any) []any {
	out := make([]any, len(list))
	for i, id := range list {
		ev := dal.NewRecord()
		ev.Set("id", id)
		out[i] = ev
	}
	return out
}

func recordIDs(rs []*dal.Record) []any {
	out := make([]any, len(rs))
	for i, r := range rs {
		out[i], _ = r.Get("id")
	}
	return out
}

// CreateSalesChannelsInput mirrors CreateSalesChannelsWorkflowInput.
type CreateSalesChannelsInput struct {
	SalesChannelsData []*dal.Record
}

// CreateSalesChannelsWorkflow mirrors createSalesChannelsWorkflow.
var CreateSalesChannelsWorkflow = composer.CreateWorkflow("create-sales-channels",
	func(e *composer.Execution, input CreateSalesChannelsInput) ([]*dal.Record, error) {
		created, err := steps.CreateSalesChannelsStep.Run(e, input.SalesChannelsData)
		if err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "sales-channel.created", Data: idEvents(recordIDs(created))}); err != nil {
			return nil, err
		}
		return created, nil
	})

// UpdateSalesChannelsWorkflow mirrors updateSalesChannelsWorkflow.
var UpdateSalesChannelsWorkflow = composer.CreateWorkflow("update-sales-channels",
	func(e *composer.Execution, input steps.UpdateSalesChannelsStepInput) ([]*dal.Record, error) {
		updated, err := steps.UpdateSalesChannelsStep.Run(e, input)
		if err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "sales-channel.updated", Data: idEvents(recordIDs(updated))}); err != nil {
			return nil, err
		}
		return updated, nil
	})

// DeleteSalesChannelsWorkflow mirrors deleteSalesChannelsWorkflow.
var DeleteSalesChannelsWorkflow = composer.CreateWorkflow("delete-sales-channels",
	func(e *composer.Execution, ids []any) (any, error) {
		if _, err := steps.CanDeleteSalesChannelsOrThrowStep.Run(e, ids); err != nil {
			return nil, err
		}
		if _, err := steps.DeleteSalesChannelsStep.Run(e, ids); err != nil {
			return nil, err
		}
		keys := dal.NewRecord()
		keys.Set("sales_channel_id", ids)
		entry := dal.NewRecord()
		entry.Set(utilsmodulessdk.SALES_CHANNEL, keys)
		if _, err := commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{entry}); err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "sales-channel.deleted", Data: idEvents(ids)}); err != nil {
			return nil, err
		}
		return nil, nil
	})

// LinkProductsInput mirrors LinkWorkflowInput.
type LinkProductsInput struct {
	ID     string
	Add    []string
	Remove []string
}

// LinkProductsToSalesChannelWorkflow mirrors linkProductsToSalesChannelWorkflow.
var LinkProductsToSalesChannelWorkflow = composer.CreateWorkflow("link-products-to-sales-channel",
	func(e *composer.Execution, input LinkProductsInput) (any, error) {
		toAdd := make([]steps.ProductChannelLink, len(input.Add))
		for i, p := range input.Add {
			toAdd[i] = steps.ProductChannelLink{SalesChannelID: input.ID, ProductID: p}
		}
		toRemove := make([]steps.ProductChannelLink, len(input.Remove))
		for i, p := range input.Remove {
			toRemove[i] = steps.ProductChannelLink{SalesChannelID: input.ID, ProductID: p}
		}
		if _, err := steps.AssociateProductsWithSalesChannelsStep.Run(e, toAdd); err != nil {
			return nil, err
		}
		if _, err := steps.DetachProductsFromSalesChannelsStep.Run(e, toRemove); err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		var events []any
		for _, id := range append(append([]string{}, input.Add...), input.Remove...) {
			if seen[id] {
				continue
			}
			seen[id] = true
			ev := dal.NewRecord()
			ev.Set("id", id)
			events = append(events, ev)
		}
		if events == nil {
			events = []any{}
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product.updated", Data: events}); err != nil {
			return nil, err
		}
		return nil, nil
	})
