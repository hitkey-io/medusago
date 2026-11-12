package giftcards

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

func filterField() *zod.Schema {
	return zod.Union(zod.String(), zod.Array(zod.String()), utils.CreateOperatorMap(nil, nil)).Optional()
}

// Validators.
var (
	// AdminGetGiftCardsParams mirrors AdminGetGiftCardsParams.
	AdminGetGiftCardsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(15), Offset: utils.Num(0)}).Merge(zod.Object(
		zod.F("q", zod.String().Optional()),
		zod.F("customer_id", filterField()),
		zod.F("id", filterField()),
		zod.F("reference_id", filterField()),
		zod.F("reference", filterField()),
		zod.F("status", filterField()),
		zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
		zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	)).Strict()
	// AdminGetGiftCardOrdersParams mirrors AdminGetGiftCardOrdersParams.
	AdminGetGiftCardOrdersParams = zod.Object(zod.F("fields", zod.String().Optional())).Strict()
	// AdminCreateGiftCard mirrors AdminCreateGiftCard.
	AdminCreateGiftCard = zod.Object(
		zod.F("status", zod.Enum("pending", "redeemed").Default("pending")),
		zod.F("currency_code", zod.String()),
		zod.F("value", zod.Number().Min(1)),
		zod.F("code", zod.String().Optional()),
		zod.F("expires_at", zod.String().Nullish()),
		zod.F("reference", zod.String().Optional()),
		zod.F("reference_id", zod.String().Optional()),
		zod.F("line_item_id", zod.String().Optional()),
		zod.F("note", zod.String().Nullish()),
		zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Optional()),
	).Strict()
	// AdminUpdateGiftCard mirrors AdminUpdateGiftCard.
	AdminUpdateGiftCard = zod.Object(
		zod.F("status", zod.Enum("pending", "redeemed").Optional()),
		zod.F("note", zod.String().Nullish()),
		zod.F("expires_at", zod.String().Nullish()),
		zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Optional()),
	).Strict()
)
