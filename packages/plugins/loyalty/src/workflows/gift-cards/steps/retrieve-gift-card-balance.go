package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	storecredit "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/store-credit"
)

// AccountStatsRetriever is the store credit module method giving the stats
// (balance, credits, debits) of an account.
type AccountStatsRetriever interface {
	RetrieveAccountStats(ctx context.Context, accountID string) (*dal.Record, error)
}

// StoreCreditModule resolves the store credit module service
// (PluginModule.STORE_CREDIT) as an AccountStatsRetriever.
func StoreCreditModule(c composer.Container) AccountStatsRetriever {
	s, _ := c.Resolve(storecredit.ServiceName).(AccountStatsRetriever)
	return s
}

// RetrieveGiftCardsBalanceStepInput mirrors RetrieveGiftCardsBalanceStepInput.
type RetrieveGiftCardsBalanceStepInput struct {
	// GiftCardStoreCreditAccountMap maps gift card ids to their store credit
	// account ids.
	GiftCardStoreCreditAccountMap *dal.Record   `json:"giftCardStoreCreditAccountMap"`
	GiftCards                     []*dal.Record `json:"giftCards"`
}

// RetrieveGiftCardsBalanceStep mirrors retrieveGiftCardsBalanceStep
// ("retrieve-gift-cards-balance"): the account stats keyed by gift card
// code.
var RetrieveGiftCardsBalanceStep = composer.CreateStep("retrieve-gift-cards-balance",
	func(ctx composer.StepExecutionContext, in RetrieveGiftCardsBalanceStepInput) (composer.StepResponse[*dal.Record], error) {
		balances := dal.NewRecord()
		module := StoreCreditModule(ctx.Container)
		for _, giftCard := range in.GiftCards {
			accountID, _ := get(in.GiftCardStoreCreditAccountMap, str(giftCard, "id")).(string)
			stats, err := module.RetrieveAccountStats(ctx.Context, accountID)
			if err != nil {
				return composer.StepResponse[*dal.Record]{}, err
			}
			balances.Set(str(giftCard, "code"), stats)
		}
		return composer.NewStepResponse(balances), nil
	}, nil)
