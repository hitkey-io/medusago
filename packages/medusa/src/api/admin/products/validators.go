// Package products mirrors api/admin/products.
package products

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	productvariants "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-variants"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

func operatorMap() *zod.Schema { return utils.CreateOperatorMap(nil, nil) }

// AdminGetProductParams mirrors AdminGetProductParams.
var AdminGetProductParams = utils.CreateSelectParams()

// AdminGetProductsParamsDirectFields mirrors AdminGetProductsParamsDirectFields.
var AdminGetProductsParamsDirectFields = zod.Object(
	zod.F("variants", productvariants.AdminGetProductVariantsParamsFields.Omit("q").
		Merge(commonvalidators.ApplyAndAndOrOperators(productvariants.AdminGetProductVariantsParamsFields.Omit("q"))).Optional()),
	zod.F("status", commonvalidators.ProductStatusEnum().Array().Optional()),
)

// AdminGetProductsParams mirrors AdminGetProductsParams (a transform: not
// strict).
var AdminGetProductsParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(50)}).
	Merge(AdminGetProductsParamsDirectFields).
	Merge(zod.Object(zod.F("price_list_id", zod.String().Array().Optional())).
		Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetProductsParamsDirectFields)).
		Merge(commonvalidators.GetProductsParams(operatorMap))).
	Transform(commonvalidators.TransformProductParams)
