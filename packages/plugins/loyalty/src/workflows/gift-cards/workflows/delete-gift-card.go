package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/gift-cards/steps"
)

// DeleteGiftCardWorkflowInput mirrors DeleteGiftCardWorkflowInput.
type DeleteGiftCardWorkflowInput struct {
	ID string `json:"id"`
}

// DeleteGiftCardWorkflow mirrors deleteGiftCardWorkflow ("delete-gift-card").
var DeleteGiftCardWorkflow = composer.CreateWorkflow("delete-gift-card",
	func(e *composer.Execution, input DeleteGiftCardWorkflowInput) (any, error) {
		if _, err := queryGraph(e, "get-gift-card-query", "gift_card", []string{"id", "status"}, filter("id", input.ID), true); err != nil {
			return nil, err
		}
		_, err := steps.DeleteGiftCardsStep.Run(e, steps.DeleteGiftCardsStepInput{ID: []any{input.ID}})
		return nil, err
	})
