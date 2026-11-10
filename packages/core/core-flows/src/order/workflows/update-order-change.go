package workflows

import (
	"fmt"
	"strings"

	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type carryPromotionsValidation struct {
	OrderChange *dal.Record
	Order       *dal.Record
	Carry       bool
}

var validateCarryPromotionsFlagStep = composer.CreateStep("validate-carry-promotions-flag",
	func(ctx composer.StepExecutionContext, in carryPromotionsValidation) (composer.StepResponse[any], error) {
		if err := ThrowIfOrderChangeIsNotActive(in.OrderChange); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if !in.Carry {
			return composer.NewStepResponse[any](nil), nil
		}
		var invalid []string
		for _, p := range recs(g(in.Order, "promotions")) {
			am := rec(g(p, "application_method"))
			if am == nil {
				continue
			}
			allocation, typ := g(am, "allocation"), g(am, "type")
			name := g(p, "code")
			if !truthy(name) {
				name = g(p, "id")
			}
			if allocation != "across" && allocation != "each" {
				return composer.StepResponse[any]{}, invalidData(fmt.Sprintf("Promotion %v has invalid allocation. Only promotions with EACH or ACROSS allocation can be carried over to outbound exchange items.", name))
			}
			if (typ == "fixed" && allocation != "each") || (typ == "percentage" && allocation != "each" && allocation != "across") {
				invalid = append(invalid, fmt.Sprint(name))
			}
		}
		if len(invalid) > 0 {
			return composer.StepResponse[any]{}, invalidData(fmt.Sprintf("Promotions with codes %s have invalid allocation. Fixed promotions must have EACH allocation, and percentage promotions must have EACH or ACROSS allocation.", strings.Join(invalid, ", ")))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// OnCarryPromotionsFlagSetInput mirrors OnCarryPromotionsFlagSetWorkflowInput.
type OnCarryPromotionsFlagSetInput struct {
	OrderChangeID any
	Carry         bool
}

// OnCarryPromotionsFlagSet mirrors onCarryPromotionsFlagSet.
var OnCarryPromotionsFlagSet = composer.CreateWorkflow("on-carry-promotions-flag-set",
	func(e *composer.Execution, in OnCarryPromotionsFlagSetInput) (any, error) {
		f := dal.NewRecord()
		f.Set("id", in.OrderChangeID)
		v := dal.NewRecord()
		v.Set("filters", f)
		orderChange, err := remoteOne(e, "order-change-query", "order_change", v, []string{"id", "status", "version", "exchange_id", "claim_id",
			"return_id", "order_id", "canceled_at", "confirmed_at", "declined_at", "carry_over_promotions"}, true)
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(orderChange, "order_id"), append(append([]string{}, FieldsToComputeAdjustmentsForPreview...), "promotions.application_method.*"))
		if err != nil {
			return nil, err
		}
		if _, err := validateCarryPromotionsFlagStep.Run(e, carryPromotionsValidation{orderChange, order, in.Carry}); err != nil {
			return nil, err
		}
		return nil, computeAdjustments(e, order, orderChange)
	})

// UpdateOrderChangeWorkflow mirrors updateOrderChangeWorkflow.
var UpdateOrderChangeWorkflow = composer.CreateWorkflow("update-order-change-workflow",
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		updated, err := ordersteps.UpdateOrderChangesStep.Run(e, []*dal.Record{in})
		if err != nil {
			return nil, err
		}
		if carry, ok := g(in, "carry_over_promotions").(bool); ok {
			if _, err := OnCarryPromotionsFlagSet.RunAsStep(e, OnCarryPromotionsFlagSetInput{OrderChangeID: g(in, "id"), Carry: carry}); err != nil {
				return nil, err
			}
		}
		if len(updated) > 0 {
			return updated[0], nil
		}
		return nil, nil
	})
