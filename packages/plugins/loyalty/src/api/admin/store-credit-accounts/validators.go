package storecreditaccounts

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

func filterField() *zod.Schema {
	return zod.Union(zod.String(), zod.Array(zod.String()), utils.CreateOperatorMap(nil, nil)).Optional()
}

// Validators.
var (
	// AdminGetStoreCreditAccountsParams mirrors AdminGetStoreCreditAccountsParams.
	AdminGetStoreCreditAccountsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(15), Offset: utils.Num(0)}).Merge(zod.Object(
		zod.F("q", zod.String().Optional()),
		zod.F("id", filterField()),
		zod.F("currency_code", filterField()),
		zod.F("customer_id", filterField()),
		zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
		zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	)).Strict()
	// AdminCreateStoreCreditAccount mirrors AdminCreateStoreCreditAccount.
	AdminCreateStoreCreditAccount = zod.Object(
		zod.F("currency_code", zod.String()),
		zod.F("customer_id", zod.String().Optional()),
		zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullable().Optional()),
	).Strict()
	// AdminCreditStoreCreditAccountParams mirrors AdminCreditStoreCreditAccountParams.
	AdminCreditStoreCreditAccountParams = zod.Object(
		zod.F("amount", zod.Number()),
		zod.F("note", zod.String().Optional()),
	)
	// AdminDebitStoreCreditAccountParams mirrors AdminDebitStoreCreditAccountParams.
	AdminDebitStoreCreditAccountParams = zod.Object(
		zod.F("amount", zod.Number()),
		zod.F("note", zod.String().Optional()),
	)
	// AdminGetStoreCreditAccountTransactionsParams mirrors
	// AdminGetStoreCreditAccountTransactionsParams.
	AdminGetStoreCreditAccountTransactionsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(15), Offset: utils.Num(0)}).Merge(zod.Object(
		zod.F("q", zod.String().Optional()),
		zod.F("id", filterField()),
		zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
		zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	)).Strict()
)
