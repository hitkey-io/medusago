// Package steps mirrors src/workflows/store-credit/steps.
package steps

import (
	cartutils "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	storecredit "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/store-credit"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/utils"
)

// Service resolves the store credit module service.
func Service(c composer.Container) *storecredit.StoreCreditModuleService {
	s, _ := c.Resolve(storecredit.ServiceName).(*storecredit.StoreCreditModuleService)
	return s
}

// ids returns the ids of the records.
func ids(list []*dal.Record) []any {
	out := make([]any, 0, len(list))
	for _, r := range list {
		v, _ := r.Get("id")
		out = append(out, v)
	}
	return out
}

func idList(v any) []any {
	list, _ := v.([]any)
	return list
}

// CreateStoreCreditAccountsStep mirrors createStoreCreditAccountsStep
// ("create-store-credit-accounts"): an account without a code gets one
// prefixed with "SC"; the compensation deletes the created accounts.
var CreateStoreCreditAccountsStep = composer.CreateStep("create-store-credit-accounts",
	func(ctx composer.StepExecutionContext, input []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		for _, account := range input {
			code, _ := account.Get("code")
			if !cartutils.IsPresent(code) {
				account.Set("code", utils.GenerateCode("SC", 4))
			}
		}
		accounts, err := Service(ctx.Container).CreateStoreCreditAccounts(ctx.Context, input)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(accounts, any(ids(accounts))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		list := idList(data)
		if len(list) == 0 {
			return nil
		}
		return Service(ctx.Container).DeleteStoreCreditAccounts(ctx.Context, list)
	})
