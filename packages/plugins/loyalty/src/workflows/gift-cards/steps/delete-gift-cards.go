package steps

import (
	"errors"

	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// DeleteGiftCardsStepInput mirrors DeleteGiftCardsStepInput.
type DeleteGiftCardsStepInput struct {
	ID []any `json:"id"`
}

// DeleteGiftCardsStep mirrors deleteGiftCardsStep ("delete-gift-cards").
// Like upstream, it calls deleteGiftCards({ id }) on the store credit module
// (PluginModule.STORE_CREDIT), whose service has no gift card methods: the
// step fails with the TypeError of the call (no route runs it).
var DeleteGiftCardsStep = composer.CreateStep("delete-gift-cards",
	func(ctx composer.StepExecutionContext, in DeleteGiftCardsStepInput) (composer.StepResponse[[]any], error) {
		return composer.StepResponse[[]any]{}, errors.New("module.deleteGiftCards is not a function")
	},
	func(ctx composer.StepExecutionContext, data any) error {
		list, _ := data.([]any)
		if len(list) == 0 {
			return nil
		}
		return errors.New("module.restoreGiftCards is not a function")
	})
