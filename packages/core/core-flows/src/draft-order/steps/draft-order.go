// Package steps mirrors @medusajs/core-flows/src/draft-order/steps.
package steps

import (
	"context"
	"fmt"

	cartsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	lockingsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/locking/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type orderOps interface {
	Internal(className string) *utilsmodulessdk.InternalService
	UpdateOrdersReturning(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	DeleteOrders(ctx context.Context, ids []any) error
}

func orderOf(c composer.Container) orderOps { return c.Resolve(utilsmodulessdk.ORDER).(orderOps) }

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func recs(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		out := make([]*dal.Record, 0, len(t))
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

// ValidateDraftOrderStep mirrors validateDraftOrderStep.
var ValidateDraftOrderStep = composer.CreateStep("validate-draft-order",
	func(ctx composer.StepExecutionContext, order *dal.Record) (composer.StepResponse[any], error) {
		draft, _ := get(order, "is_draft_order").(bool)
		if get(order, "status") != "draft" && !draft {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Order %v is not a draft order", get(order, "id")))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// DeleteDraftOrdersStep mirrors deleteDraftOrdersStep.
var DeleteDraftOrdersStep = composer.CreateStep("delete-draft-orders",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		return composer.NewStepResponse[any](nil), orderOf(ctx.Container).DeleteOrders(ctx.Context, ids)
	}, nil)

// LineItemAdjustmentsInput mirrors CreateDraftOrderLineItemAdjustmentsStepInput.
type LineItemAdjustmentsInput struct {
	OrderID     any
	Adjustments []*dal.Record
	Version     any
}

// CreateDraftOrderLineItemAdjustmentsStep mirrors createDraftOrderLineItemAdjustmentsStep.
var CreateDraftOrderLineItemAdjustmentsStep = composer.CreateStep("create-draft-order-line-item-adjustments",
	func(ctx composer.StepExecutionContext, in LineItemAdjustmentsInput) (composer.StepResponse[any], error) {
		if len(in.Adjustments) == 0 {
			return composer.NewStepResponse[any](nil, any([]any{})), nil
		}
		var data []*dal.Record
		for _, a := range in.Adjustments {
			if v := get(a, "amount"); v == nil || v == 0 || v == float64(0) {
				continue
			}
			r := dal.NewRecord()
			for _, k := range a.Keys() {
				v, _ := a.Get(k)
				r.Set(k, v)
			}
			r.Set("version", in.Version)
			r.Set("order_id", in.OrderID)
			data = append(data, r)
		}
		created, err := orderOf(ctx.Container).Internal("OrderLineItemAdjustment").Create(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		ids := commonsteps.RecordIDs(created)
		return composer.NewStepResponse[any](ids, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := orderOf(ctx.Container).Internal("OrderLineItemAdjustment").Delete(ctx.Context, ids)
		return err
	})

// CreateDraftOrderShippingMethodAdjustmentsStep mirrors createDraftOrderShippingMethodAdjustmentsStep.
var CreateDraftOrderShippingMethodAdjustmentsStep = composer.CreateStep("create-draft-order-shipping-method-adjustments",
	func(ctx composer.StepExecutionContext, adjustments []*dal.Record) (composer.StepResponse[any], error) {
		if len(adjustments) == 0 {
			return composer.NewStepResponse[any](nil, any([]any{})), nil
		}
		created, err := orderOf(ctx.Container).Internal("OrderShippingMethodAdjustment").Create(ctx.Context, adjustments)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		ids := commonsteps.RecordIDs(created)
		return composer.NewStepResponse[any](ids, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := orderOf(ctx.Container).Internal("OrderShippingMethodAdjustment").Delete(ctx.Context, ids)
		return err
	})

func removeAdjustmentsStep(name, className string) *composer.Step[[]any, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
			if len(ids) == 0 {
				return composer.NewStepResponse[any](nil, any([]any{})), nil
			}
			if _, err := orderOf(ctx.Container).Internal(className).Delete(ctx.Context, ids); err != nil {
				return composer.StepResponse[any]{}, err
			}
			return composer.NewStepResponse[any](nil, any(ids)), nil
		},
		func(ctx composer.StepExecutionContext, data any) error {
			ids, _ := data.([]any)
			if len(ids) == 0 {
				return nil
			}
			_, err := orderOf(ctx.Container).Internal(className).Restore(ctx.Context, ids)
			return err
		})
}

// RemoveDraftOrderLineItemAdjustmentsStep mirrors removeDraftOrderLineItemAdjustmentsStep.
var RemoveDraftOrderLineItemAdjustmentsStep = removeAdjustmentsStep("remove-draft-order-line-item-adjustments", "OrderLineItemAdjustment")

// RemoveDraftOrderShippingMethodAdjustmentsStep mirrors removeDraftOrderShippingMethodAdjustmentsStep.
var RemoveDraftOrderShippingMethodAdjustmentsStep = removeAdjustmentsStep("remove-draft-order-shipping-method-adjustments", "OrderShippingMethodAdjustment")

// UpdateDraftOrderPromotionsInput mirrors UpdateDraftOrderPromotionsStepInput.
type UpdateDraftOrderPromotionsInput struct {
	ID         any
	PromoCodes []string
	Action     string
}

func orderPromoLink(orderID, promotionID any) *dal.Record {
	o := dal.NewRecord()
	o.Set("order_id", orderID)
	p := dal.NewRecord()
	p.Set("promotion_id", promotionID)
	l := dal.NewRecord()
	l.Set(utilsmodulessdk.ORDER, o)
	l.Set(utilsmodulessdk.PROMOTION, p)
	return l
}

type promoRevert struct {
	created, dismissed []*dal.Record
}

// UpdateDraftOrderPromotionsStep mirrors updateDraftOrderPromotionsStep.
var UpdateDraftOrderPromotionsStep = composer.CreateStep("update-draft-order-promotions",
	func(ctx composer.StepExecutionContext, in UpdateDraftOrderPromotionsInput) (composer.StepResponse[any], error) {
		action := in.Action
		if action == "" {
			action = cartsteps.PromotionActionAdd
		}
		v := dal.NewRecord()
		v.Set("order_id", []any{in.ID})
		existing, err := commonsteps.RemoteQuery(ctx, "order_promotion", v, []string{"order_id", "promotion_id"})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		existingMap := map[any]bool{}
		for _, l := range recs(existing) {
			existingMap[get(l, "promotion_id")] = true
		}
		var toCreate, toDismiss []*dal.Record
		if len(in.PromoCodes) > 0 {
			codes := make([]any, len(in.PromoCodes))
			for i, c := range in.PromoCodes {
				codes[i] = c
			}
			f := dal.NewRecord()
			f.Set("code", codes)
			promotions, err := commonsteps.ModuleOf(ctx.Container, utilsmodulessdk.PROMOTION).ListBySuffix(ctx.Context, "Promotions", f, utilsmodulessdk.FindConfig{Select: []string{"id"}})
			if err != nil {
				return composer.StepResponse[any]{}, err
			}
			for _, p := range promotions {
				link := orderPromoLink(in.ID, get(p, "id"))
				if action == cartsteps.PromotionActionAdd || action == cartsteps.PromotionActionReplace {
					toCreate = append(toCreate, link)
				}
				if action == cartsteps.PromotionActionRemove && existingMap[get(p, "id")] {
					toDismiss = append(toDismiss, link)
				}
			}
		}
		if action == cartsteps.PromotionActionReplace {
			for _, l := range recs(existing) {
				toDismiss = append(toDismiss, orderPromoLink(get(l, "order_id"), get(l, "promotion_id")))
			}
		}
		link := commonsteps.LinkOf(ctx.Container)
		if len(toDismiss) > 0 {
			if _, err := link.Dismiss(ctx.Context, toDismiss); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		if len(toCreate) > 0 {
			if _, err := link.Create(ctx.Context, toCreate); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		return composer.NewStepResponse[any](nil, any(promoRevert{toCreate, toDismiss})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		r, ok := data.(promoRevert)
		if !ok {
			return nil
		}
		link := commonsteps.LinkOf(ctx.Container)
		if len(r.dismissed) > 0 {
			if _, err := link.Create(ctx.Context, r.dismissed); err != nil {
				return err
			}
		}
		if len(r.created) > 0 {
			if _, err := link.Dismiss(ctx.Context, r.created); err != nil {
				return err
			}
		}
		return nil
	})

// RefreshDraftOrderAdjustmentsInput mirrors RefreshDraftOrderAdjustmentsWorkflowInput.
type RefreshDraftOrderAdjustmentsInput struct {
	Order      *dal.Record
	PromoCodes []string
	Action     string
	Version    any
}

// RefreshDraftOrderAdjustmentsWorkflow mirrors refreshDraftOrderAdjustmentsWorkflow
// (defined with the steps so that order workflows can run it without an
// import cycle).
var RefreshDraftOrderAdjustmentsWorkflow = composer.CreateWorkflow("refresh-draft-order-adjustments",
	func(e *composer.Execution, in RefreshDraftOrderAdjustmentsInput) (any, error) {
		key, _ := get(in.Order, "id").(string)
		if _, err := lockingsteps.AcquireLockStep.Run(e, lockingsteps.AcquireLockStepInput{Key: lockingsteps.Key(key), Timeout: 2, TTL: 10}); err != nil {
			return nil, err
		}
		hookIn := dal.NewRecord()
		hookIn.Set("order", in.Order)
		hookIn.Set("promo_codes", in.PromoCodes)
		hookIn.Set("action", in.Action)
		if _, err := composer.CreateHookWithResult(e, "setPromotionContext", hookIn); err != nil {
			return nil, err
		}
		codes, err := cartsteps.GetPromotionCodesToApply.Run(e, cartsteps.PromotionCodesInput{Cart: in.Order, PromoCodes: in.PromoCodes, Action: in.Action})
		if err != nil {
			return nil, err
		}
		actions, err := cartsteps.GetActionsToComputeFromPromotionsStep.Run(e, cartsteps.ActionsInput{ComputeActionContext: in.Order, PromotionCodesToApply: codes})
		if err != nil {
			return nil, err
		}
		prepared, err := cartsteps.PrepareAdjustmentsFromPromotionActionsStep.Run(e, actions)
		if err != nil {
			return nil, err
		}
		err = composer.Parallelize(
			func() error {
				_, err := RemoveDraftOrderLineItemAdjustmentsStep.Run(e, prepared.LineItemAdjustmentIDsToRemove)
				return err
			},
			func() error {
				_, err := RemoveDraftOrderShippingMethodAdjustmentsStep.Run(e, prepared.ShippingMethodAdjustmentIDsToRemove)
				return err
			},
			func() error {
				_, err := CreateDraftOrderLineItemAdjustmentsStep.Run(e, LineItemAdjustmentsInput{OrderID: get(in.Order, "id"), Adjustments: prepared.LineItemAdjustmentsToCreate, Version: in.Version})
				return err
			},
			func() error {
				_, err := CreateDraftOrderShippingMethodAdjustmentsStep.Run(e, prepared.ShippingMethodAdjustmentsToCreate)
				return err
			},
			func() error {
				_, err := UpdateDraftOrderPromotionsStep.Run(e, UpdateDraftOrderPromotionsInput{ID: get(in.Order, "id"), PromoCodes: in.PromoCodes, Action: in.Action})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		_, err = lockingsteps.ReleaseLockStep.Run(e, lockingsteps.ReleaseLockStepInput{Key: lockingsteps.Key(key)})
		return nil, err
	})

// UpdateDraftOrderStepInput mirrors UpdateDraftOrderStepInput.
type UpdateDraftOrderStepInput struct {
	Order *dal.Record
	Input *dal.Record
}

// UpdateDraftOrderStep mirrors updateDraftOrderStep.
var UpdateDraftOrderStep = composer.CreateStep("update-draft-order",
	func(ctx composer.StepExecutionContext, in UpdateDraftOrderStepInput) (composer.StepResponse[*dal.Record], error) {
		u := dal.NewRecord()
		u.Set("id", get(in.Order, "id"))
		for _, k := range in.Input.Keys() {
			v, _ := in.Input.Get(k)
			u.Set(k, v)
		}
		updated, err := orderOf(ctx.Container).UpdateOrdersReturning(ctx.Context, []*dal.Record{u})
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		var out *dal.Record
		if len(updated) > 0 {
			out = updated[0]
		}
		return composer.NewStepResponse(out, any(in.Order)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.(*dal.Record)
		if prev == nil {
			return nil
		}
		_, err := orderOf(ctx.Container).UpdateOrdersReturning(ctx.Context, []*dal.Record{prev})
		return err
	})

// ConvertDraftOrderStep mirrors convertDraftOrderStep.
var ConvertDraftOrderStep = composer.CreateStep("convert-draft-order",
	func(ctx composer.StepExecutionContext, id any) (composer.StepResponse[*dal.Record], error) {
		u := dal.NewRecord()
		u.Set("id", id)
		u.Set("status", "pending")
		u.Set("is_draft_order", false)
		updated, err := orderOf(ctx.Container).UpdateOrdersReturning(ctx.Context, []*dal.Record{u})
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		var out *dal.Record
		if len(updated) > 0 {
			out = updated[0]
		}
		return composer.NewStepResponse(out, id), nil
	},
	func(ctx composer.StepExecutionContext, id any) error {
		if id == nil {
			return nil
		}
		u := dal.NewRecord()
		u.Set("id", id)
		u.Set("status", "draft")
		u.Set("is_draft_order", true)
		_, err := orderOf(ctx.Container).UpdateOrdersReturning(ctx.Context, []*dal.Record{u})
		return err
	})
