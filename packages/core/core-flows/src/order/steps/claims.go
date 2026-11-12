package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type claimOps interface {
	Internal(className string) *utilsmodulessdk.InternalService
	CreateOrderClaims(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	CreateOrderClaimItems(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	DeleteOrderClaimItems(ctx context.Context, ids []any) error
	CreateOrderLineItems(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	DeleteOrderLineItems(ctx context.Context, ids []any) error
}

func claimsOf(c composer.Container) claimOps { return c.Resolve(utilsmodulessdk.ORDER).(claimOps) }

// CreateOrderClaimsStep mirrors createOrderClaimsStep.
var CreateOrderClaimsStep = composer.CreateStep("create-order-claims",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := claimsOf(ctx.Container).CreateOrderClaims(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(ids(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		list, _ := data.([]any)
		if list == nil {
			return nil
		}
		_, err := claimsOf(ctx.Container).Internal("OrderClaim").Delete(ctx.Context, list)
		return err
	})

// CreateClaimItemsFromActionsInput mirrors CreateOrderClaimItemsFromActionsInput.
type CreateClaimItemsFromActionsInput struct {
	Changes []*dal.Record
	ClaimID any
}

// CreateOrderClaimItemsFromActionsStep mirrors createOrderClaimItemsFromActionsStep.
var CreateOrderClaimItemsFromActionsStep = composer.CreateStep("create-claim-items-from-change-actions",
	func(ctx composer.StepExecutionContext, in CreateClaimItemsFromActionsInput) (composer.StepResponse[[]*dal.Record], error) {
		items := make([]*dal.Record, 0, len(in.Changes))
		for _, c := range in.Changes {
			details, _ := get(c, "details").(*dal.Record)
			r := dal.NewRecord()
			r.Set("claim_id", in.ClaimID)
			r.Set("item_id", get(details, "reference_id"))
			r.Set("quantity", get(details, "quantity"))
			if v, ok := c.Get("internal_note"); ok {
				r.Set("note", v)
			}
			if m := get(details, "metadata"); m != nil {
				r.Set("metadata", m)
			} else {
				r.Set("metadata", dal.NewRecord())
			}
			switch get(c, "action") {
			case "ITEM_ADD":
				r.Set("is_additional_item", true)
			case "WRITE_OFF_ITEM":
				if details != nil {
					if v, ok := details.Get("reason"); ok {
						r.Set("reason", v)
					}
				}
			}
			items = append(items, r)
		}
		created, err := claimsOf(ctx.Container).CreateOrderClaimItems(ctx.Context, items)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(ids(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		list, _ := data.([]any)
		if list == nil {
			return nil
		}
		return claimsOf(ctx.Container).DeleteOrderClaimItems(ctx.Context, list)
	})

// UpdateOrderClaimsStep mirrors updateOrderClaimsStep.
var UpdateOrderClaimsStep = updateEntitiesStep("update-order-claim", "OrderClaim", "OrderClaims")

// UpdateOrderChangesStep mirrors updateOrderChangesStep.
var UpdateOrderChangesStep = updateEntitiesStep("update-order-changes", "OrderChange", "OrderChanges")

// DeleteClaimsStep mirrors deleteClaimsStep.
var DeleteClaimsStep = softDeleteStep("delete-claims", "OrderClaim")

// CreateOrderLineItemsStep mirrors createOrderLineItemsStep.
var CreateOrderLineItemsStep = composer.CreateStep("create-order-line-items-step",
	func(ctx composer.StepExecutionContext, items []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(items) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]any{})), nil
		}
		created, err := claimsOf(ctx.Container).CreateOrderLineItems(ctx.Context, items)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(ids(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		list, _ := data.([]any)
		if len(list) == 0 {
			return nil
		}
		return claimsOf(ctx.Container).DeleteOrderLineItems(ctx.Context, list)
	})

// ListOrderChangeActionsByTypeInput mirrors ListOrderChangeActionsByTypeStepInput.
type ListOrderChangeActionsByTypeInput struct {
	OrderChangeID any
	ActionType    string
}

// ListOrderChangeActionsByTypeStep mirrors listOrderChangeActionsByTypeStep.
var ListOrderChangeActionsByTypeStep = composer.CreateStep("list-order-change-actions-by-type",
	func(ctx composer.StepExecutionContext, in ListOrderChangeActionsByTypeInput) (composer.StepResponse[[]any], error) {
		f := dal.NewRecord()
		f.Set("order_change_id", in.OrderChangeID)
		f.Set("action", in.ActionType)
		rows, err := claimsOf(ctx.Container).Internal("OrderChangeAction").List(ctx.Context, f, utilsmodulessdk.FindConfig{Select: []string{"id"}})
		if err != nil {
			return composer.StepResponse[[]any]{}, err
		}
		return composer.NewStepResponse(ids(rows)), nil
	}, nil)

// DeleteOrderChangeActionsStep mirrors deleteOrderChangeActionsStep.
var DeleteOrderChangeActionsStep = softDeleteStep("delete-order-change-actions", "OrderChangeAction")

// UpdateOrderChangeActionsStep mirrors updateOrderChangeActionsStep: the
// actions are read before the update (the fields of the input, metadata and
// details as object fields, plus their order/return/claim/exchange ids,
// version and order change id) and the compensation writes them back.
var UpdateOrderChangeActionsStep = composer.CreateStep("update-order-change-actions",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(data) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
		}
		svc := returnsOf(ctx.Container)
		objs := make([]common.ObjectLike, len(data))
		for i, d := range data {
			objs[i] = d
		}
		selects, relations := common.GetSelectsAndRelationsFromObjectArray(objs, common.SelectsAndRelationsOptions{ObjectFields: []string{"metadata", "details"}}, "")
		selects = append(selects, "order_id", "return_id", "claim_id", "exchange_id", "version", "order_change_id")
		f := dal.NewRecord()
		f.Set("id", ids(data))
		before, err := svc.Internal("OrderChangeAction").List(ctx.Context, f, utilsmodulessdk.FindConfig{Select: dedupe(selects), Relations: relations})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		in := make([]any, len(data))
		for i, d := range data {
			in[i] = d
		}
		res, err := svc.Internal("OrderChangeAction").Update(ctx.Context, in)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		updated, _ := res.([]*dal.Record)
		return composer.NewStepResponse(updated, any(before)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		before, _ := data.([]*dal.Record)
		if len(before) == 0 {
			return nil
		}
		in := make([]any, len(before))
		for i, d := range before {
			in[i] = d
		}
		_, err := returnsOf(ctx.Container).Internal("OrderChangeAction").Update(ctx.Context, in)
		return err
	})

// UpdateOrderShippingMethodsStep mirrors updateOrderShippingMethodsStep.
var UpdateOrderShippingMethodsStep = updateEntitiesStep("update-order-shopping-methods", "OrderShippingMethod", "OrderShippingMethods")
