package carts

import (
	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	commonworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/workflows"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/loyalty"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/utils"
	gcsteps "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/gift-cards/steps"
)

// RefreshCartGiftCardsWorkflowInput mirrors RefreshCartGiftCardsWorkflowInput.
type RefreshCartGiftCardsWorkflowInput struct {
	CartID string `json:"cart_id"`
}

// RefreshCartGiftCardsWorkflow mirrors refreshCartGiftCardsWorkflow
// ("refresh-cart-gift-card"): the gift card credit lines (and their links)
// are removed and created again from the current balances and cart total;
// expired gift cards are dropped.
var RefreshCartGiftCardsWorkflow = composer.CreateWorkflow("refresh-cart-gift-card",
	func(e *composer.Execution, input RefreshCartGiftCardsWorkflowInput) ([]any, error) {
		existingRows, err := gcQueryGraph(e, "get-existing-cart-query", "cart", []string{
			"id", "currency_code", "total", "gift_cards.code", "credit_lines.id", "credit_lines.reference", "credit_lines.reference_id",
		}, gcRecord("id", input.CartID), true)
		if err != nil {
			return nil, err
		}
		existingCart := gcFirst(existingRows)
		creditLineIDs, giftCardIDs := []any{}, []any{}
		for _, creditLine := range gcRecords(gcGet(existingCart, "credit_lines")) {
			if gcStr(creditLine, "reference") == "gift-card" {
				creditLineIDs = append(creditLineIDs, gcGet(creditLine, "id"))
				giftCardIDs = append(giftCardIDs, gcGet(creditLine, "reference_id"))
			}
		}
		if _, err := cartworkflows.DeleteCartCreditLinesWorkflow.RunAsStep(e, cartworkflows.DeleteCartCreditLinesInput{ID: creditLineIDs}); err != nil {
			return nil, err
		}
		toDismiss := make([]*dal.Record, len(giftCardIDs))
		for i, id := range giftCardIDs {
			toDismiss[i] = gcRecord(
				utilsmodulessdk.CART, gcRecord("cart_id", gcGet(existingCart, "id")),
				loyalty.ServiceName, gcRecord("gift_card_id", id),
			)
		}
		if _, err := commonworkflows.DismissLinksWorkflow.RunAsStep(e, toDismiss); err != nil {
			return nil, err
		}

		codes := []any{}
		for _, gc := range gcRecords(gcGet(existingCart, "gift_cards")) {
			codes = append(codes, gcGet(gc, "code"))
		}
		giftCardRows, err := gcQueryGraph(e, "get-gift-card-query", "gift_card", []string{"id", "code", "status", "currency_code", "expires_at"}, gcRecord("code", codes), false)
		if err != nil {
			return nil, err
		}
		// expired gift cards are dropped rather than rejected, so that a
		// cart holding a gift card that expired after it was applied can
		// still be refreshed (their credit lines and links are gone)
		giftCards := []*dal.Record{}
		for _, gc := range gcRecords(giftCardRows) {
			if !utils.IsGiftCardExpired(gcGet(gc, "expires_at")) {
				giftCards = append(giftCards, gc)
			}
		}
		ids := make([]any, len(giftCards))
		for i, gc := range giftCards {
			ids[i] = gcGet(gc, "id")
		}
		linkRows, err := gcQueryGraph(e, "get-gift-card-store-credit-account-query", "gift_card_store_credit_account",
			[]string{"id", "store_credit_account_id", "gift_card_id"}, gcRecord("gift_card_id", ids), false)
		if err != nil {
			return nil, err
		}
		accountMap := dal.NewRecord()
		for _, link := range gcRecords(linkRows) {
			accountMap.Set(gcStr(link, "gift_card_id"), gcGet(link, "store_credit_account_id"))
		}
		balances, err := gcsteps.RetrieveGiftCardsBalanceStep.Run(e, gcsteps.RetrieveGiftCardsBalanceStepInput{GiftCards: giftCards, GiftCardStoreCreditAccountMap: accountMap})
		if err != nil {
			return nil, err
		}
		if _, err := ValidateGiftCardBalancesStep.Run(e, ValidateGiftCardBalancesStepInput{GiftCards: giftCards, GiftCardsBalanceMap: balances}); err != nil {
			return nil, err
		}
		cartRows, err := gcQueryGraph(e, "get-cart-query", "cart", []string{"id", "currency_code", "total", "gift_cards.code", "customer_id"}, gcRecord("id", input.CartID), true)
		if err != nil {
			return nil, err
		}
		cart := gcFirst(cartRows)
		toCreate, err := gcCreditLinesForGiftCards(cart, giftCards, balances, true)
		if err != nil {
			return nil, err
		}
		creditLines, err := cartworkflows.CreateCartCreditLinesWorkflow.RunAsStep(e, toCreate)
		if err != nil {
			return nil, err
		}
		links := make([]*dal.Record, len(giftCards))
		for i, gc := range giftCards {
			links[i] = gcRecord(
				utilsmodulessdk.CART, gcRecord("cart_id", gcGet(cart, "id")),
				loyalty.ServiceName, gcRecord("gift_card_id", gcGet(gc, "id")),
			)
		}
		if _, err := commonworkflows.CreateLinksWorkflow.RunAsStep(e, links); err != nil {
			return nil, err
		}
		return creditLines, nil
	})
