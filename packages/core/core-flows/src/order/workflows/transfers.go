package workflows

import (
	"crypto/rand"
	"fmt"
	"time"

	cartsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// randomUUID mirrors crypto.randomUUID() (version 4).
func randomUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func notAllowed(msg string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, msg)
}

func graphFirst(e *composer.Execution, name, entity string, filters *dal.Record, fields []string, throw bool) (*dal.Record, error) {
	res, err := commonsteps.UseQueryGraphStep.Config(name, composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
		GraphInput: query.GraphInput{Entity: entity, Fields: fields, Filters: filters},
		Options:    joiner.Options{ThrowIfKeyNotFound: throw},
	})
	if err != nil {
		return nil, err
	}
	if list := recs(res.Data); len(list) > 0 {
		return list[0], nil
	}
	return nil, nil
}

func transferChange(in *dal.Record, orderID any) *dal.Record {
	c := dal.NewRecord()
	c.Set("change_type", "transfer")
	c.Set("order_id", orderID)
	if v := g(in, "logged_in_user"); v != nil {
		c.Set("created_by", v)
	}
	setIfPresent(c, in, "description", "internal_note")
	return c
}

func transferAction(change *dal.Record, orderID, customerID any, details *dal.Record) *dal.Record {
	a := dal.NewRecord()
	a.Set("order_change_id", g(change, "id"))
	a.Set("order_id", orderID)
	a.Set("action", "TRANSFER_CUSTOMER")
	a.Set("version", g(change, "version"))
	a.Set("reference", "customer")
	a.Set("reference_id", customerID)
	a.Set("details", details)
	return a
}

type transferValidation struct {
	Order    *dal.Record
	Customer *dal.Record
}

var requestOrderTransferValidationStep = composer.CreateStep("request-order-transfer-validation",
	func(ctx composer.StepExecutionContext, in transferValidation) (composer.StepResponse[any], error) {
		if err := throwIfOrderIsCancelled(in.Order); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if !truthy(g(in.Customer, "has_account")) {
			return composer.StepResponse[any]{}, invalidData(fmt.Sprintf("Cannot transfer order: %v to a guest customer account: %v", g(in.Order, "id"), g(in.Customer, "email")))
		}
		if g(in.Order, "customer_id") == g(in.Customer, "id") {
			return composer.StepResponse[any]{}, invalidData(fmt.Sprintf("Order: %v already belongs to customer: %v", g(in.Order, "id"), g(in.Customer, "id")))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// RequestOrderTransferWorkflow mirrors requestOrderTransferWorkflow.
var RequestOrderTransferWorkflow = composer.CreateWorkflow("request-order-transfer-workflow",
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		orderID := g(in, "order_id")
		order, err := remoteOne(e, "", "orders", idVars(orderID), []string{"id", "email", "status", "customer_id"}, true)
		if err != nil {
			return nil, err
		}
		customer, err := remoteOne(e, "customer-query", "customers", idVars(g(in, "customer_id")), []string{"id", "email", "has_account"}, true)
		if err != nil {
			return nil, err
		}
		if _, err := requestOrderTransferValidationStep.Run(e, transferValidation{order, customer}); err != nil {
			return nil, err
		}
		change, err := ordersteps.CreateOrderChangeStep.Run(e, transferChange(in, orderID))
		if err != nil {
			return nil, err
		}
		d := dal.NewRecord()
		d.Set("token", randomUUID())
		d.Set("original_email", g(order, "email"))
		if truthy(g(in, "update_order_email")) {
			d.Set("new_email", g(customer, "email"))
		}
		if _, err := CreateOrderChangeActionsWorkflow.RunAsStep(e, []*dal.Record{transferAction(change, orderID, g(in, "customer_id"), d)}); err != nil {
			return nil, err
		}
		u := dal.NewRecord()
		u.Set("id", g(change, "id"))
		u.Set("status", "requested")
		if v := g(in, "logged_in_user"); v != nil {
			u.Set("requested_by", v)
		}
		u.Set("requested_at", time.Now())
		if _, err := ordersteps.UpdateOrderChangesStep.Run(e, []*dal.Record{u}); err != nil {
			return nil, err
		}
		ev := dal.NewRecord()
		ev.Set("id", orderID)
		ev.Set("order_change_id", g(change, "id"))
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "order.transfer_requested", Data: ev}); err != nil {
			return nil, err
		}
		return previewFor(e, orderID)
	})

type tokenValidation struct {
	Token       any
	Order       *dal.Record
	OrderChange *dal.Record
}

func transferToken(orderChange *dal.Record) any {
	for _, a := range recs(g(orderChange, "actions")) {
		if g(a, "action") == "TRANSFER_CUSTOMER" {
			return g(rec(g(a, "details")), "token")
		}
	}
	return nil
}

var acceptOrderTransferValidationStep = composer.CreateStep("accept-order-transfer-validation",
	func(ctx composer.StepExecutionContext, in tokenValidation) (composer.StepResponse[any], error) {
		if err := throwIfOrderIsCancelled(in.Order); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if in.OrderChange == nil || g(in.OrderChange, "change_type") != "transfer" {
			return composer.StepResponse[any]{}, invalidData(fmt.Sprintf("Order %v does not have an order transfer request.", g(in.Order, "id")))
		}
		token, _ := in.Token.(string)
		if token == "" || any(token) != transferToken(in.OrderChange) {
			return composer.StepResponse[any]{}, notAllowed("Invalid token.")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// AcceptOrderTransferWorkflow mirrors acceptOrderTransferWorkflow.
var AcceptOrderTransferWorkflow = composer.CreateWorkflow("accept-order-transfer-workflow",
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		orderID := g(in, "order_id")
		f := dal.NewRecord()
		f.Set("id", orderID)
		order, err := graphFirst(e, "order-query", "order", f, []string{"id", "email", "status", "customer_id"}, true)
		if err != nil {
			return nil, err
		}
		cf := dal.NewRecord()
		cf.Set("order_id", orderID)
		cf.Set("status", []any{"requested"})
		orderChange, err := graphFirst(e, "order-change-query", "order_change", cf, []string{"id", "status", "change_type", "actions.id", "actions.order_id",
			"actions.action", "actions.details", "actions.reference", "actions.reference_id", "actions.internal_note"}, false)
		if err != nil {
			return nil, err
		}
		if _, err := acceptOrderTransferValidationStep.Run(e, tokenValidation{g(in, "token"), order, orderChange}); err != nil {
			return nil, err
		}
		if _, err := ordersteps.ConfirmOrderChangesStep.Run(e, ordersteps.ConfirmOrderChangesInput{OrderID: g(order, "id"), Changes: []*dal.Record{orderChange}}); err != nil {
			return nil, err
		}
		return previewFor(e, orderID)
	})

func activeChangeGraph(e *composer.Execution, orderID any) (*dal.Record, error) {
	cf := dal.NewRecord()
	cf.Set("order_id", orderID)
	cf.Set("status", []any{"pending", "requested"})
	return graphFirst(e, "order-change-query", "order_change", cf, []string{"id", "status", "version", "actions.*"}, false)
}

var cancelTransferOrderRequestValidationStep = composer.CreateStep("validate-cancel-transfer-order-request",
	func(ctx composer.StepExecutionContext, in struct {
		Order       *dal.Record
		OrderChange *dal.Record
		Input       *dal.Record
	}) (composer.StepResponse[any], error) {
		if err := throwIfIsCancelled(in.Order, "Order"); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := ThrowIfOrderChangeIsNotActive(in.OrderChange); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if g(in.Input, "actor_type") == "user" {
			return composer.NewStepResponse[any](nil), nil
		}
		var refID any
		for _, a := range recs(g(in.OrderChange, "actions")) {
			if g(a, "action") == "TRANSFER_CUSTOMER" {
				refID = g(a, "reference_id")
				break
			}
		}
		if refID != g(in.Input, "logged_in_user_id") {
			return composer.StepResponse[any]{}, notAllowed("This customer is not allowed to cancel the transfer.")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// CancelOrderTransferRequestWorkflow mirrors cancelOrderTransferRequestWorkflow.
var CancelOrderTransferRequestWorkflow = composer.CreateWorkflow("cancel-transfer-order-request",
	func(e *composer.Execution, in *dal.Record) (any, error) {
		f := dal.NewRecord()
		f.Set("id", g(in, "order_id"))
		order, err := graphFirst(e, "order-query", "order", f, []string{"id", "version", "canceled_at"}, true)
		if err != nil {
			return nil, err
		}
		orderChange, err := activeChangeGraph(e, g(in, "order_id"))
		if err != nil {
			return nil, err
		}
		if _, err := cancelTransferOrderRequestValidationStep.Run(e, struct {
			Order       *dal.Record
			OrderChange *dal.Record
			Input       *dal.Record
		}{order, orderChange, in}); err != nil {
			return nil, err
		}
		_, err = ordersteps.DeleteOrderChangesStep.Run(e, []any{g(orderChange, "id")})
		return nil, err
	})

var declineTransferOrderRequestValidationStep = composer.CreateStep("validate-decline-transfer-order-request",
	func(ctx composer.StepExecutionContext, in tokenValidation) (composer.StepResponse[any], error) {
		if err := throwIfIsCancelled(in.Order, "Order"); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := ThrowIfOrderChangeIsNotActive(in.OrderChange); err != nil {
			return composer.StepResponse[any]{}, err
		}
		token, _ := in.Token.(string)
		if token == "" || transferToken(in.OrderChange) != any(token) {
			return composer.StepResponse[any]{}, notAllowed("Invalid token.")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// DeclineOrderTransferRequestWorkflow mirrors declineOrderTransferRequestWorkflow.
var DeclineOrderTransferRequestWorkflow = composer.CreateWorkflow("decline-transfer-order-request",
	func(e *composer.Execution, in *dal.Record) (any, error) {
		f := dal.NewRecord()
		f.Set("id", g(in, "order_id"))
		order, err := graphFirst(e, "order-query", "order", f, []string{"id", "version", "declined_at"}, true)
		if err != nil {
			return nil, err
		}
		orderChange, err := activeChangeGraph(e, g(in, "order_id"))
		if err != nil {
			return nil, err
		}
		if _, err := declineTransferOrderRequestValidationStep.Run(e, tokenValidation{g(in, "token"), order, orderChange}); err != nil {
			return nil, err
		}
		d := dal.NewRecord()
		d.Set("id", g(orderChange, "id"))
		_, err = ordersteps.DeclineOrderChangeStep.Run(e, d)
		return nil, err
	})

var transferOrderToGuestValidationStep = composer.CreateStep("transfer-order-to-guest-validation",
	func(ctx composer.StepExecutionContext, in transferValidation) (composer.StepResponse[any], error) {
		if err := throwIfOrderIsCancelled(in.Order); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if g(in.Order, "customer_id") == g(in.Customer, "id") {
			return composer.StepResponse[any]{}, invalidData(fmt.Sprintf("Order: %v already belongs to customer: %v", g(in.Order, "id"), g(in.Customer, "id")))
		}
		if truthy(g(in.Customer, "has_account")) {
			return composer.StepResponse[any]{}, invalidData(fmt.Sprintf("Cannot transfer order: %v to registered customer account: %v. Only guest customers are allowed.", g(in.Order, "id"), g(in.Customer, "email")))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// TransferOrderToGuestWorkflow mirrors transferOrderToGuestWorkflow.
var TransferOrderToGuestWorkflow = composer.CreateWorkflow("transfer-order-to-guest-workflow",
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		orderID := g(in, "order_id")
		f := dal.NewRecord()
		f.Set("id", orderID)
		order, err := graphFirst(e, "order-query", "order", f, []string{"id", "email", "status", "customer_id"}, true)
		if err != nil {
			return nil, err
		}
		customerData, err := cartsteps.FindOrCreateCustomerStep.Run(e, cartsteps.FindOrCreateCustomerInput{Email: g(in, "email")})
		if err != nil {
			return nil, err
		}
		customer := customerData.Customer
		if _, err := transferOrderToGuestValidationStep.Run(e, transferValidation{order, customer}); err != nil {
			return nil, err
		}
		change, err := ordersteps.CreateOrderChangeStep.Run(e, transferChange(in, orderID))
		if err != nil {
			return nil, err
		}
		d := dal.NewRecord()
		d.Set("original_email", g(order, "email"))
		d.Set("new_email", g(customer, "email"))
		if _, err := CreateOrderChangeActionsWorkflow.RunAsStep(e, []*dal.Record{transferAction(change, orderID, g(customer, "id"), d)}); err != nil {
			return nil, err
		}
		if _, err := ordersteps.ConfirmOrderChangesStep.Run(e, ordersteps.ConfirmOrderChangesInput{OrderID: orderID, Changes: []*dal.Record{change}, ConfirmedBy: g(in, "logged_in_user")}); err != nil {
			return nil, err
		}
		return previewFor(e, orderID)
	})
