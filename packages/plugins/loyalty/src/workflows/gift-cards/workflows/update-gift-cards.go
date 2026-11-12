package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/gift-cards/steps"
)

// UpdateGiftCardsWorkflow mirrors updateGiftCardsWorkflow
// ("update-gift-cards-workflow").
var UpdateGiftCardsWorkflow = composer.CreateWorkflow("update-gift-cards-workflow",
	func(e *composer.Execution, input []*dal.Record) ([]*dal.Record, error) {
		return steps.UpdateGiftCardsStep.Run(e, input)
	})
