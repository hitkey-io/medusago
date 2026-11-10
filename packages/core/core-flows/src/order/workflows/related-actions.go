package workflows

import (
	"context"
	"errors"
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	reservationsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/reservation/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// relatedKind describes a claim or an exchange for the shared action
// workflows (order/workflows/claim and order/workflows/exchange).
type relatedKind struct {
	entry string // remote query entry point
	label string // throwIfIsCancelled label
	key   string // claim_id / exchange_id
	word  string // claim / exchange (messages)
}

var (
	claimKind    = relatedKind{"order_claim", "Claim", "claim_id", "claim"}
	exchangeKind = relatedKind{"order_exchange", "Exchange", "exchange_id", "exchange"}
)

// RelatedActionInput mirrors the { claim_id | exchange_id, action_id, data }
// workflow inputs.
type RelatedActionInput struct {
	ID       any
	ActionID any
	Data     *dal.Record
}

type relatedValidation struct {
	Order       *dal.Record
	Entity      *dal.Record
	OrderChange *dal.Record
	Input       RelatedActionInput
}

// relatedActionValidationStep mirrors the validation steps of the claim and
// exchange action workflows; medusaErr selects MedusaError INVALID_DATA
// over a plain Error.
func relatedActionValidationStep(name string, k relatedKind, withOrder, medusaErr bool, missing, wrong string, allowed ...string) *composer.Step[relatedValidation, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, in relatedValidation) (composer.StepResponse[any], error) {
			fail := func(err error) (composer.StepResponse[any], error) { return composer.StepResponse[any]{}, err }
			if withOrder {
				if err := throwIfIsCancelled(in.Order, "Order"); err != nil {
					return fail(err)
				}
			}
			if err := throwIfIsCancelled(in.Entity, k.label); err != nil {
				return fail(err)
			}
			if err := ThrowIfOrderChangeIsNotActive(in.OrderChange); err != nil {
				return fail(err)
			}
			mk := func(msg string) error {
				if medusaErr {
					return invalidData(msg)
				}
				return errors.New(msg)
			}
			a := FindChangeAction(in.OrderChange, in.Input.ActionID)
			if a == nil {
				return fail(mk(fmt.Sprintf(missing, in.Input.ID, g(in.OrderChange, "id"))))
			}
			act, _ := g(a, "action").(string)
			for _, x := range allowed {
				if x == act {
					return composer.NewStepResponse[any](nil), nil
				}
			}
			return fail(mk(fmt.Sprintf(wrong, g(a, "id"))))
		}, nil)
}

func queryRelated(e *composer.Execution, k relatedKind, id any, fields []string) (*dal.Record, error) {
	return remoteOne(e, "", k.entry, idVars(id), fields, true)
}

func relatedChange(e *composer.Execution, k relatedKind, entity *dal.Record, fields []string) (*dal.Record, error) {
	return activeOrderChangeFor(e, g(entity, "order_id"), k.key, g(entity, "id"), fields)
}

func refreshRelatedShipping(e *composer.Execution, k relatedKind, orderChange, entity *dal.Record) error {
	if k.key == "claim_id" {
		args := dal.NewRecord()
		args.Set("order_change_id", g(orderChange, "id"))
		args.Set("claim_id", g(entity, "id"))
		args.Set("order_id", g(entity, "order_id"))
		_, err := RefreshClaimShippingWorkflow.RunAsStep(e, args)
		return err
	}
	return refreshExchangeShipping(e, orderChange, entity)
}

func updateActionQuantity(e *composer.Execution, in RelatedActionInput, orderChange *dal.Record) error {
	original := rec(g(FindChangeAction(orderChange, in.ActionID), "details"))
	u := dal.NewRecord()
	u.Set("id", in.ActionID)
	d := dal.NewRecord()
	if q := g(in.Data, "quantity"); q != nil {
		d.Set("quantity", q)
	} else if q := g(original, "quantity"); q != nil {
		d.Set("quantity", q)
	}
	u.Set("details", d)
	setIfPresent(u, in.Data, "internal_note")
	_, err := ordersteps.UpdateOrderChangeActionsStep.Run(e, []*dal.Record{u})
	return err
}

func baseRelated(e *composer.Execution, k relatedKind, id any, orderFields, changeFields []string) (entity, order, orderChange *dal.Record, err error) {
	if entity, err = queryRelated(e, k, id, []string{"id", "status", "order_id", "canceled_at"}); err != nil {
		return
	}
	if orderFields != nil {
		if order, err = queryOrderOne(e, g(entity, "order_id"), orderFields); err != nil {
			return
		}
	}
	orderChange, err = relatedChange(e, k, entity, changeFields)
	return
}

var updateClaimItemValidationStep = relatedActionValidationStep("update-claim-item-validation", claimKind, true, false,
	"No request claim found for claim %v in order change %v", "Action %v is not claiming the item", "WRITE_OFF_ITEM")

// UpdateClaimItemWorkflow mirrors updateClaimItemWorkflow.
var UpdateClaimItemWorkflow = composer.CreateWorkflow("update-claim-item",
	func(e *composer.Execution, in RelatedActionInput) (*dal.Record, error) {
		claim, order, orderChange, err := baseRelated(e, claimKind, in.ID, []string{"id", "status", "canceled_at", "items.*"}, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if _, err := updateClaimItemValidationStep.Run(e, relatedValidation{order, claim, orderChange, in}); err != nil {
			return nil, err
		}
		if err := updateActionQuantity(e, in, orderChange); err != nil {
			return nil, err
		}
		return previewFor(e, g(order, "id"))
	})

var removeClaimItemActionValidationStep = relatedActionValidationStep("remove-item-claim-action-validation", claimKind, true, false,
	"No item claim found for claim %v in order change %v", "Action %v is not claiming an item", "WRITE_OFF_ITEM")

// RemoveItemClaimActionWorkflow mirrors removeItemClaimActionWorkflow.
var RemoveItemClaimActionWorkflow = composer.CreateWorkflow("remove-item-claim-action",
	func(e *composer.Execution, in RelatedActionInput) (*dal.Record, error) {
		claim, order, orderChange, err := baseRelated(e, claimKind, in.ID, []string{"id", "status", "canceled_at", "items.*"}, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if _, err := removeClaimItemActionValidationStep.Run(e, relatedValidation{order, claim, orderChange, in}); err != nil {
			return nil, err
		}
		if _, err := ordersteps.DeleteOrderChangeActionsStep.Run(e, []any{in.ActionID}); err != nil {
			return nil, err
		}
		if err := refreshRelatedShipping(e, claimKind, orderChange, claim); err != nil {
			return nil, err
		}
		return previewFor(e, g(order, "id"))
	})

func addItemUpdateWorkflow(name string, k relatedKind, validation *composer.Step[relatedValidation, any], changeFields []string) *composer.Workflow[RelatedActionInput, *dal.Record] {
	return composer.CreateWorkflow(name,
		func(e *composer.Execution, in RelatedActionInput) (*dal.Record, error) {
			entity, order, orderChange, err := baseRelated(e, k, in.ID, append(append([]string{}, FieldsToComputeAdjustmentsForPreview...), "status", "canceled_at"), changeFields)
			if err != nil {
				return nil, err
			}
			if _, err := validation.Run(e, relatedValidation{order, entity, orderChange, in}); err != nil {
				return nil, err
			}
			if err := updateActionQuantity(e, in, orderChange); err != nil {
				return nil, err
			}
			if err := computeAdjustments(e, order, orderChange); err != nil {
				return nil, err
			}
			if err := refreshRelatedShipping(e, k, orderChange, entity); err != nil {
				return nil, err
			}
			return previewFor(e, g(order, "id"))
		})
}

// UpdateClaimAddItemWorkflow mirrors updateClaimAddItemWorkflow.
var UpdateClaimAddItemWorkflow = addItemUpdateWorkflow("update-claim-add-item", claimKind,
	relatedActionValidationStep("update-claim-add-item-validation", claimKind, true, true,
		"No request to add item for claim %v in order change %v", "Action %v is not adding an item", "ITEM_ADD"),
	[]string{"id", "status", "version", "actions.*", "claim_id", "carry_over_promotions"})

// UpdateExchangeAddItemWorkflow mirrors updateExchangeAddItemWorkflow.
var UpdateExchangeAddItemWorkflow = addItemUpdateWorkflow("update-exchange-add-item", exchangeKind,
	relatedActionValidationStep("update-exchange-add-item-validation", exchangeKind, true, true,
		"No request to add item for exchange %v in order change %v", "Action %v is not adding an item", "ITEM_ADD"),
	[]string{"id", "status", "version", "exchange_id", "actions.*", "carry_over_promotions"})

// removeAddItemWorkflow mirrors removeAddItemClaimActionWorkflow and
// removeItemExchangeActionWorkflow: when no item is added anymore, the
// outbound shipping method goes too (the exchange otherwise refreshes its
// shipping).
func removeAddItemWorkflow(name string, k relatedKind, validation *composer.Step[relatedValidation, any], updatedFields []string, refreshOtherwise bool, removeShipping func() *composer.Workflow[RelatedActionInput, *dal.Record]) *composer.Workflow[RelatedActionInput, *dal.Record] {
	return composer.CreateWorkflow(name,
		func(e *composer.Execution, in RelatedActionInput) (*dal.Record, error) {
			entity, order, orderChange, err := baseRelated(e, k, in.ID, []string{"id", "status", "canceled_at", "items.*"}, []string{"id", "status", "version", "actions.*"})
			if err != nil {
				return nil, err
			}
			if _, err := validation.Run(e, relatedValidation{order, entity, orderChange, in}); err != nil {
				return nil, err
			}
			if _, err := ordersteps.DeleteOrderChangeActionsStep.Run(e, []any{in.ActionID}); err != nil {
				return nil, err
			}
			f := dal.NewRecord()
			f.Set("order_id", g(entity, "order_id"))
			f.Set(k.key, g(entity, "id"))
			f.Set("status", []any{"pending", "requested"})
			v := dal.NewRecord()
			v.Set("filters", f)
			updated, err := remoteOne(e, "updated-order-change-query", "order_change", v, updatedFields, false)
			if err != nil {
				return nil, err
			}
			var toDelete any
			hasItems := false
			for _, a := range recs(g(updated, "actions")) {
				if g(a, "action") == "ITEM_ADD" {
					hasItems = true
				}
			}
			if !hasItems {
				for _, a := range recs(g(updated, "actions")) {
					if g(a, "action") == "SHIPPING_ADD" && g(a, k.key) == g(entity, "id") && !truthy(g(a, "return_id")) {
						toDelete = g(a, "id")
						break
					}
				}
			}
			if truthy(toDelete) {
				if _, err := removeShipping().RunAsStep(e, RelatedActionInput{ID: g(entity, "id"), ActionID: toDelete}); err != nil {
					return nil, err
				}
			} else if refreshOtherwise {
				if err := refreshRelatedShipping(e, k, orderChange, entity); err != nil {
					return nil, err
				}
			}
			return previewFor(e, g(order, "id"))
		})
}

// RemoveAddItemClaimActionWorkflow mirrors removeAddItemClaimActionWorkflow.
var RemoveAddItemClaimActionWorkflow = removeAddItemWorkflow("remove-item-claim-add-action", claimKind,
	relatedActionValidationStep("remove-item-claim-add-action-validation", claimKind, true, false,
		"No item claim found for claim %v in order change %v", "Action %v is not adding an item", "ITEM_ADD"),
	[]string{"actions.action", "actions.claim_id", "actions.id", "actions.return_id"}, false,
	func() *composer.Workflow[RelatedActionInput, *dal.Record] { return RemoveClaimShippingMethodWorkflow })

// RemoveItemExchangeActionWorkflow mirrors removeItemExchangeActionWorkflow.
var RemoveItemExchangeActionWorkflow = removeAddItemWorkflow("remove-item-exchange-action", exchangeKind,
	relatedActionValidationStep("remove-item-exchange-action-validation", exchangeKind, true, false,
		"No item exchange found for exchange %v in order change %v", "Action %v is not adding an item", "ITEM_ADD"),
	[]string{"actions.action", "actions.id", "actions.exchange_id", "actions.return_id"}, true,
	func() *composer.Workflow[RelatedActionInput, *dal.Record] {
		return RemoveExchangeShippingMethodWorkflow
	})

func updateRelatedShippingWorkflow(name string, k relatedKind, validation *composer.Step[relatedValidation, any]) *composer.Workflow[RelatedActionInput, *dal.Record] {
	return composer.CreateWorkflow(name,
		func(e *composer.Execution, in RelatedActionInput) (*dal.Record, error) {
			entity, err := queryRelated(e, k, in.ID, []string{"id", "status", "order_id", "canceled_at", "order.currency_code"})
			if err != nil {
				return nil, err
			}
			orderChange, err := relatedChange(e, k, entity, []string{"id", "status", "version", "actions.*"})
			if err != nil {
				return nil, err
			}
			hookIn := dal.NewRecord()
			hookIn.Set("order_"+k.word, entity)
			hookIn.Set("order_change", orderChange)
			hookIn.Set("additional_data", g(in.Data, "additional_data"))
			if err := composer.CreateHook(e, "setPricingContext", hookIn); err != nil {
				return nil, err
			}
			var options []*dal.Record
			if customAmountNull(in.Data) {
				original := FindChangeAction(orderChange, in.ActionID)
				if options, err = FetchShippingOptionsForMethod(e, g(original, "reference_id"), g(rec(g(entity, "order")), "currency_code")); err != nil {
					return nil, err
				}
			}
			if _, err := validation.Run(e, relatedValidation{Entity: entity, OrderChange: orderChange, Input: in}); err != nil {
				return nil, err
			}
			if err := UpdateShippingMethodAction(e, OrderChangeActionInput{OrderID: in.ID, ActionID: in.ActionID, Data: in.Data}, orderChange, options); err != nil {
				return nil, err
			}
			return previewFor(e, g(entity, "order_id"))
		})
}

func removeRelatedShippingWorkflow(name string, k relatedKind, validation *composer.Step[relatedValidation, any]) *composer.Workflow[RelatedActionInput, *dal.Record] {
	return composer.CreateWorkflow(name,
		func(e *composer.Execution, in RelatedActionInput) (*dal.Record, error) {
			entity, _, orderChange, err := baseRelated(e, k, in.ID, nil, []string{"id", "status", "version", "actions.*"})
			if err != nil {
				return nil, err
			}
			if _, err := validation.Run(e, relatedValidation{Entity: entity, OrderChange: orderChange, Input: in}); err != nil {
				return nil, err
			}
			if err := RemoveShippingMethodAction(e, orderChange, in.ActionID); err != nil {
				return nil, err
			}
			return previewFor(e, g(entity, "order_id"))
		})
}

// UpdateClaimShippingMethodWorkflow mirrors updateClaimShippingMethodWorkflow.
var UpdateClaimShippingMethodWorkflow = updateRelatedShippingWorkflow("update-claim-shipping-method", claimKind,
	relatedActionValidationStep("validate-update-claim-shipping-method", claimKind, false, false,
		"No shipping method found for claim %v in order change %v", "Action %v is not adding a shipping method", "SHIPPING_ADD"))

// RemoveClaimShippingMethodWorkflow mirrors removeClaimShippingMethodWorkflow.
var RemoveClaimShippingMethodWorkflow = removeRelatedShippingWorkflow("remove-claim-shipping-method", claimKind,
	relatedActionValidationStep("validate-remove-claim-shipping-method", claimKind, false, false,
		"No shipping method found for claim %v in order change %v", "Action %v is not adding a shipping method", "SHIPPING_ADD"))

// UpdateExchangeShippingMethodWorkflow mirrors updateExchangeShippingMethodWorkflow.
var UpdateExchangeShippingMethodWorkflow = updateRelatedShippingWorkflow("update-exchange-shipping-method", exchangeKind,
	relatedActionValidationStep("validate-update-exchange-shipping-method", exchangeKind, false, false,
		"No shipping method found for exchange %v in order change %v", "Action %v is not adding a shipping method", "SHIPPING_ADD"))

// RemoveExchangeShippingMethodWorkflow mirrors removeExchangeShippingMethodWorkflow.
var RemoveExchangeShippingMethodWorkflow = removeRelatedShippingWorkflow("remove-exchange-shipping-method", exchangeKind,
	relatedActionValidationStep("validate-remove-exchange-shipping-method", exchangeKind, false, false,
		"No shipping method found for exchange %v in order change %v", "Action %v is not adding a shipping method", "SHIPPING_ADD"))

type relatedCanceler interface {
	CancelClaim(ctx context.Context, data *dal.Record) (*dal.Record, error)
	CancelExchange(ctx context.Context, data *dal.Record) (*dal.Record, error)
}

func cancelRelatedStep(name string, cancel func(svc relatedCanceler, ctx context.Context, data *dal.Record) error) *composer.Step[*dal.Record, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, data *dal.Record) (composer.StepResponse[any], error) {
			svc, _ := ctx.Container.Resolve(utilsmodulessdk.ORDER).(relatedCanceler)
			if err := cancel(svc, ctx.Context, data); err != nil {
				return composer.StepResponse[any]{}, err
			}
			return composer.NewStepResponse[any](nil, g(data, "order_id")), nil
		}, ordersteps.RevertLastVersion)
}

// CancelOrderClaimStep mirrors cancelOrderClaimStep.
var CancelOrderClaimStep = cancelRelatedStep("cancel-order-claim", func(svc relatedCanceler, ctx context.Context, data *dal.Record) error {
	_, err := svc.CancelClaim(ctx, data)
	return err
})

// CancelOrderExchangeStep mirrors cancelOrderExchangeStep.
var CancelOrderExchangeStep = cancelRelatedStep("cancel-order-swap", func(svc relatedCanceler, ctx context.Context, data *dal.Record) error {
	_, err := svc.CancelExchange(ctx, data)
	return err
})

func cancelRelatedValidation(name string, k relatedKind, message string) *composer.Step[[2]*dal.Record, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, in [2]*dal.Record) (composer.StepResponse[any], error) {
			if err := throwIfIsCancelled(in[0], k.label); err != nil {
				return composer.StepResponse[any]{}, err
			}
			for _, f := range recs(g(in[1], "fulfillments")) {
				if !truthy(g(f, "canceled_at")) {
					return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, message)
				}
			}
			return composer.NewStepResponse[any](nil), nil
		}, nil)
}

func cancelRelatedWorkflow(name string, k relatedKind, validation *composer.Step[[2]*dal.Record, any], cancel *composer.Step[*dal.Record, any], queryName string) *composer.Workflow[*dal.Record, any] {
	return composer.CreateWorkflow(name,
		func(e *composer.Execution, in *dal.Record) (any, error) {
			entity, err := remoteOne(e, queryName, k.entry, idVars(g(in, k.key)), []string{"id", "order_id", "return_id", "canceled_at", "additional_items.item_id"}, true)
			if err != nil {
				return nil, err
			}
			orderReturn, err := remoteOne(e, "return-query", "return", idVars(g(entity, "return_id")), []string{"id", "fulfillments.canceled_at"}, false)
			if err != nil {
				return nil, err
			}
			if _, err := validation.Run(e, [2]*dal.Record{entity, orderReturn}); err != nil {
				return nil, err
			}
			var lineItemIDs []any
			for _, i := range recs(g(entity, "additional_items")) {
				lineItemIDs = append(lineItemIDs, g(i, "item_id"))
			}
			d := dal.NewRecord()
			d.Set(k.key, g(entity, "id"))
			d.Set("order_id", g(entity, "order_id"))
			if v, ok := in.Get("canceled_by"); ok {
				d.Set("canceled_by", v)
			}
			var deleted []any
			err = composer.Parallelize(
				func() error { _, err := cancel.Run(e, d); return err },
				func() error {
					var err error
					deleted, err = reservationsteps.DeleteReservationsByLineItemsStep.Run(e, lineItemIDs)
					return err
				},
			)
			if err != nil {
				return nil, err
			}
			events := []any{}
			for _, id := range deleted {
				ev := dal.NewRecord()
				ev.Set("id", id)
				ev.Set("order_id", g(entity, "order_id"))
				events = append(events, ev)
			}
			if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "reservation-item.deleted", Data: events}); err != nil {
				return nil, err
			}
			if truthy(g(entity, "return_id")) {
				r := dal.NewRecord()
				r.Set("return_id", g(entity, "return_id"))
				if v, ok := in.Get("no_notification"); ok {
					r.Set("no_notification", v)
				}
				if _, err := CancelReturnWorkflow.RunAsStep(e, r); err != nil {
					return nil, err
				}
			}
			return nil, nil
		})
}

// CancelOrderClaimWorkflow mirrors cancelOrderClaimWorkflow.
var CancelOrderClaimWorkflow = cancelRelatedWorkflow("cancel-claim", claimKind,
	cancelRelatedValidation("validate-claim", claimKind, "All fulfillments must be canceled before canceling a claim"), CancelOrderClaimStep, "claim-query")

// CancelOrderExchangeWorkflow mirrors cancelOrderExchangeWorkflow.
var CancelOrderExchangeWorkflow = cancelRelatedWorkflow("cancel-exchange", exchangeKind,
	cancelRelatedValidation("validate-exchange", exchangeKind, "All fulfillments must be canceled before canceling am exchange"), CancelOrderExchangeStep, "")
