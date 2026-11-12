package storecreditaccounts

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// Validators.
var (
	// StoreGetStoreCreditAccountParams mirrors StoreGetStoreCreditAccountParams.
	StoreGetStoreCreditAccountParams = zod.Object()
	// StoreClaimStoreCreditAccountParams mirrors StoreClaimStoreCreditAccountParams.
	StoreClaimStoreCreditAccountParams = zod.Object(zod.F("code", zod.String()))
	// StoreGetStoreCreditAccountsParams mirrors StoreGetStoreCreditAccountsParams.
	StoreGetStoreCreditAccountsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(15), Offset: utils.Num(0)}).Merge(zod.Object(
		zod.F("currency_code", zod.String()),
		zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
		zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	)).Strict()
)
