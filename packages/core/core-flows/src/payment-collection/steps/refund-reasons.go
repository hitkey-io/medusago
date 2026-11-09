// Package steps mirrors @medusajs/core-flows/src/payment-collection/steps.
package steps

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateRefundReasonStep mirrors createRefundReasonStep.
var CreateRefundReasonStep = commonsteps.CreateRecordsStep("create-refund-reason", utilsmodulessdk.PAYMENT, "RefundReason")

// DeleteRefundReasonsStep mirrors deleteRefundReasonsStep.
var DeleteRefundReasonsStep = commonsteps.SoftDeleteRecordsStep("delete-refund-reasons", utilsmodulessdk.PAYMENT, "RefundReason")

// UpdateRefundReasonsStep mirrors updateRefundReasonsStep.
var UpdateRefundReasonsStep = composer.CreateStep("update-refund-reasons",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(data) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
		}
		svc := commonsteps.ModuleOf(ctx.Container, utilsmodulessdk.PAYMENT)
		f := dal.NewRecord()
		f.Set("id", commonsteps.RecordIDs(data))
		prev, err := commonsteps.PrevDataFor(ctx.Context, svc, "RefundReasons", f, data...)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		in := make([]any, len(data))
		for i, d := range data {
			in[i] = d
		}
		out, err := commonsteps.UpdateRecords(ctx.Context, svc, "RefundReason", in)
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
		svc := commonsteps.ModuleOf(ctx.Container, utilsmodulessdk.PAYMENT)
		for _, r := range prev {
			if _, err := commonsteps.UpdateRecords(ctx.Context, svc, "RefundReason", []any{r}); err != nil {
				return err
			}
		}
		return nil
	})
