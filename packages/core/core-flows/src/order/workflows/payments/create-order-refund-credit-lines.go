// Package payments mirrors @medusajs/core-flows/src/order/workflows/payments.
package payments

import (
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func g(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func recs(v any) []*dal.Record {
	switch l := v.(type) {
	case []*dal.Record:
		return l
	case []any:
		var out []*dal.Record
		for _, x := range l {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

// CreateOrderRefundCreditLinesInput mirrors the workflow input.
type CreateOrderRefundCreditLinesInput struct {
	OrderID     any
	Amount      any
	Reference   any
	ReferenceID any
	CreatedBy   any
}

var validateOrderRefundCreditLinesStep = composer.CreateStep("validate-order-refund-credit-lines",
	func(ctx composer.StepExecutionContext, order *dal.Record) (composer.StepResponse[any], error) {
		if g(order, "status") == "canceled" {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Order with id %v has been canceled.", g(order, "id")))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// CreateOrderRefundCreditLinesWorkflow mirrors createOrderRefundCreditLinesWorkflow.
var CreateOrderRefundCreditLinesWorkflow = composer.CreateWorkflow("create-order-refund-credit-lines",
	func(e *composer.Execution, in CreateOrderRefundCreditLinesInput) (any, error) {
		f := dal.NewRecord()
		f.Set("id", in.OrderID)
		res, err := commonsteps.UseQueryGraphStep.Config("get-order", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "orders", Fields: []string{"id", "status", "summary", "total", "payment_collections.id"}, Filters: f},
			Options:    joiner.Options{ThrowIfKeyNotFound: true},
		})
		if err != nil {
			return nil, err
		}
		var order *dal.Record
		if list := recs(res.Data); len(list) > 0 {
			order = list[0]
		}
		if _, err := validateOrderRefundCreditLinesStep.Run(e, order); err != nil {
			return nil, err
		}
		changeIn := dal.NewRecord()
		changeIn.Set("change_type", "credit_line")
		changeIn.Set("order_id", in.OrderID)
		if in.CreatedBy != nil {
			changeIn.Set("created_by", in.CreatedBy)
		}
		change, err := ordersteps.CreateOrderChangeStep.Run(e, changeIn)
		if err != nil {
			return nil, err
		}
		reference := in.Reference
		if reference == nil {
			reference = "payment_collection"
		}
		referenceID := in.ReferenceID
		if referenceID == nil {
			if pcs := recs(g(order, "payment_collections")); len(pcs) > 0 {
				referenceID = g(pcs[0], "id")
			}
		}
		action := dal.NewRecord()
		action.Set("order_change_id", g(change, "id"))
		action.Set("order_id", g(order, "id"))
		action.Set("version", g(change, "version"))
		action.Set("action", "CREDIT_LINE_ADD")
		action.Set("reference", reference)
		action.Set("reference_id", referenceID)
		action.Set("amount", in.Amount)
		if _, err := createOrderChangeActionsWorkflow.RunAsStep(e, []*dal.Record{action}); err != nil {
			return nil, err
		}
		cf := dal.NewRecord()
		cf.Set("order_id", in.OrderID)
		cf.Set("status", []any{"pending"})
		changes, err := commonsteps.UseQueryGraphStep.Config("order-change-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "order_change", Fields: []string{"id", "status", "change_type", "actions.id", "actions.order_id", "actions.action",
				"actions.details", "actions.reference", "actions.reference_id", "actions.internal_note"}, Filters: cf},
		})
		if err != nil {
			return nil, err
		}
		var orderChange *dal.Record
		if list := recs(changes.Data); len(list) > 0 {
			orderChange = list[0]
		}
		_, err = ordersteps.ConfirmOrderChangesStep.Run(e, ordersteps.ConfirmOrderChangesInput{OrderID: g(order, "id"), Changes: []*dal.Record{orderChange}})
		return nil, err
	})

var createOrderChangeActionsWorkflow = composer.CreateWorkflow("create-order-change-actions",
	func(e *composer.Execution, in []*dal.Record) ([]*dal.Record, error) {
		return ordersteps.AddOrderActionsStep.Run(e, in)
	})
