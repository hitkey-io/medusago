// Package steps mirrors @medusajs/core-flows/src/return-reason/steps.
package steps

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateReturnReasonsStep mirrors createReturnReasonsStep.
var CreateReturnReasonsStep = commonsteps.CreateRecordsStep("create-return-reasons", utilsmodulessdk.ORDER, "ReturnReason")

// DeleteReturnReasonStep mirrors deleteReturnReasonStep.
var DeleteReturnReasonStep = commonsteps.SoftDeleteRecordsStep("delete-return-reasons", utilsmodulessdk.ORDER, "ReturnReason")

// UpdateReturnReasonsStepInput mirrors UpdateReturnReasonStepInput.
type UpdateReturnReasonsStepInput struct {
	Selector *dal.Record
	Update   *dal.Record
}

// UpdateReturnReasonsStep mirrors updateReturnReasonsStep.
var UpdateReturnReasonsStep = composer.CreateStep("update-return-reasons",
	func(ctx composer.StepExecutionContext, in UpdateReturnReasonsStepInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := commonsteps.ModuleOf(ctx.Container, utilsmodulessdk.ORDER)
		prev, err := commonsteps.PrevDataFor(ctx.Context, svc, "ReturnReasons", in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		out, err := commonsteps.UpdateBySelector(ctx.Context, svc, "ReturnReason", in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(out, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if prev == nil {
			return nil
		}
		svc := commonsteps.ModuleOf(ctx.Container, utilsmodulessdk.ORDER)
		for _, c := range prev {
			n := dal.NewRecord()
			for _, k := range []string{"id", "value", "label", "description", "metadata"} {
				v, _ := c.Get(k)
				n.Set(k, v)
			}
			if _, err := commonsteps.UpdateRecords(ctx.Context, svc, "ReturnReason", []any{n}); err != nil {
				return err
			}
		}
		return nil
	})
