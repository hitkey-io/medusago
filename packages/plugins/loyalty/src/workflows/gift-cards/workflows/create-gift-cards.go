package workflows

import (
	commonworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/workflows"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/loyalty"
	storecredit "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/store-credit"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/gift-cards/steps"
	scsteps "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/store-credit/steps"
	scworkflows "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/store-credit/workflows"
)

// CreateGiftCardsWorkflow mirrors createGiftCardsWorkflow
// ("create-gift-cards"): the gift cards, an anonymous store credit account
// per card (linked to it and credited with its value), and the cards marked
// as redeemed.
var CreateGiftCardsWorkflow = composer.CreateWorkflow("create-gift-cards",
	func(e *composer.Execution, input []*dal.Record) ([]*dal.Record, error) {
		giftCards, err := steps.CreateGiftCardsStep.Run(e, input)
		if err != nil {
			return nil, err
		}

		// anonymous credit accounts for the gift cards, linked to them
		currencies := make([]*dal.Record, len(giftCards))
		for i, giftCard := range giftCards {
			currencies[i] = record("currency_code", get(giftCard, "currency_code"))
		}
		accounts, err := scsteps.CreateStoreCreditAccountsStep.Run(e, currencies)
		if err != nil {
			return nil, err
		}
		links := make([]*dal.Record, len(giftCards))
		for i, giftCard := range giftCards {
			links[i] = record(
				loyalty.ServiceName, record("gift_card_id", get(giftCard, "id")),
				storecredit.ServiceName, record("store_credit_account_id", get(accounts[i], "id")),
			)
		}
		if _, err := commonworkflows.CreateLinksWorkflow.RunAsStep(e, links); err != nil {
			return nil, err
		}

		// credit the accounts with the gift card values
		credits := make([]*dal.Record, len(giftCards))
		for i, giftCard := range giftCards {
			credits[i] = record(
				"account_id", get(accounts[i], "id"),
				"amount", get(giftCard, "value"),
				"note", "Gift card redemption",
				"reference", "gift_card",
				"reference_id", get(giftCard, "id"),
			)
		}
		if _, err := scworkflows.CreditAccountsWorkflow.RunAsStep(e, credits); err != nil {
			return nil, err
		}

		// mark the gift cards as redeemed
		updates := make([]*dal.Record, len(giftCards))
		for i, giftCard := range giftCards {
			updates[i] = record("id", get(giftCard, "id"), "status", "redeemed")
		}
		return UpdateGiftCardsWorkflow.RunAsStep(e, updates)
	})
