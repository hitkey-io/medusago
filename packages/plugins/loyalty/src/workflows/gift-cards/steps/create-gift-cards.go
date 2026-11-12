package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/utils"
)

// isPresent mirrors isPresent for the values of a gift card code.
func isPresent(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case []any:
		return len(t) > 0
	case *dal.Record:
		return t != nil && t.Len() > 0
	}
	return true
}

// CreateGiftCardsStep mirrors createGiftCardsStep ("create-gift-cards"): the
// gift cards are created with a generated code when none is given (the
// prefix and sections of the plugin options); the compensation deletes
// them.
var CreateGiftCardsStep = composer.CreateStep("create-gift-cards",
	func(ctx composer.StepExecutionContext, input []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		module := loyaltyModule(ctx.Container)
		options, err := module.GetOptions(ctx.Context)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		for _, giftCard := range input {
			if !isPresent(get(giftCard, "code")) {
				giftCard.Set("code", utils.GenerateCodeOpt(options["prefix"], options["sections"]))
			}
		}
		out, err := module.Call(ctx.Context, "createGiftCards", []any{toAny(input)})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		giftCards := asRecords(out)
		if giftCards == nil {
			giftCards = []*dal.Record{}
		}
		return composer.NewStepResponse(giftCards, any(ids(giftCards))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		list, _ := data.([]any)
		if len(list) == 0 {
			return nil
		}
		_, err := loyaltyModule(ctx.Container).Call(ctx.Context, "deleteGiftCards", []any{list})
		return err
	})
