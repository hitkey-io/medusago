package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/store-credit/steps"
)

// CreditAccountsWorkflow mirrors creditAccountsWorkflow ("credit-accounts"):
// records { account_id, amount, note?, reference, reference_id }.
var CreditAccountsWorkflow = composer.CreateWorkflow("credit-accounts",
	func(e *composer.Execution, input []*dal.Record) ([]*dal.Record, error) {
		return steps.CreditAccountStep.Run(e, input)
	})

// DebitAccountsWorkflow mirrors debitAccountsWorkflow ("debit-accounts").
var DebitAccountsWorkflow = composer.CreateWorkflow("debit-accounts",
	func(e *composer.Execution, input []*dal.Record) ([]*dal.Record, error) {
		return steps.DebitAccountStep.Run(e, input)
	})
