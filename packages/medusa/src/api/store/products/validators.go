package products

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

func operatorMap() *zod.Schema { return utils.CreateOperatorMap(nil, nil) }

func optionFilter() *zod.Schema {
	return zod.Object(zod.F("value", zod.String().Optional()), zod.F("option_id", zod.String().Optional()))
}

// StoreGetProductVariantsParamsFields mirrors StoreGetProductVariantsParamsFields.
var StoreGetProductVariantsParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("sku", utils.StringOrStrings().Optional()),
	zod.F("ean", utils.StringOrStrings().Optional()),
	zod.F("upc", utils.StringOrStrings().Optional()),
	zod.F("barcode", utils.StringOrStrings().Optional()),
	zod.F("options", optionFilter().Optional()),
	zod.F("created_at", operatorMap().Optional()),
	zod.F("updated_at", operatorMap().Optional()),
	zod.F("deleted_at", operatorMap().Optional()),
)

// StoreGetProductParams mirrors StoreGetProductParams.
var StoreGetProductParams = utils.CreateSelectParams().Merge(zod.Object(
	zod.F("region_id", zod.String().Optional()),
	zod.F("country_code", zod.String().Optional()),
	zod.F("province", zod.String().Optional()),
	zod.F("cart_id", zod.String().Optional()),
))

// StoreGetProductsParamsFields mirrors StoreGetProductsParamsFields.
var StoreGetProductsParamsFields = zod.Object(
	zod.F("region_id", zod.String().Optional()),
	zod.F("country_code", zod.String().Optional()),
	zod.F("province", zod.String().Optional()),
	zod.F("cart_id", zod.String().Optional()),
).Merge(commonvalidators.GetProductsParams(operatorMap)).Strict()

// StoreGetProductsParams mirrors StoreGetProductsParams.
var StoreGetProductsParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(50)}).
	Merge(StoreGetProductsParamsFields).
	Merge(zod.Object(
		zod.F("variants", zod.Object(zod.F("options", optionFilter().Optional())).
			Merge(commonvalidators.ApplyAndAndOrOperators(StoreGetProductVariantsParamsFields)).Optional()),
	).Merge(commonvalidators.ApplyAndAndOrOperators(commonvalidators.StoreGetProductParamsDirectFields(operatorMap))).Strict()).
	Transform(commonvalidators.RecursivelyNormalizeSchema(commonvalidators.TransformProductParams))
