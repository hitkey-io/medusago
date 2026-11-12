package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreditAccountStep mirrors creditAccountStep ("credit-account"): the
// compensation deletes the created transactions.
var CreditAccountStep = composer.CreateStep("credit-account",
	func(ctx composer.StepExecutionContext, input []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		transactions, err := Service(ctx.Container).CreditAccounts(ctx.Context, input)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(transactions, any(ids(transactions))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		list := idList(data)
		if len(list) == 0 {
			return nil
		}
		return Service(ctx.Container).DeleteTransactions(ctx.Context, list)
	})
