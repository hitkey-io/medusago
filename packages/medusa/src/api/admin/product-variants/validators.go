// Package productvariants mirrors api/admin/product-variants.
package productvariants

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// AdminGetProductVariantsParamsFields mirrors AdminGetProductVariantsParamsFields.
var AdminGetProductVariantsParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("manage_inventory", commonvalidators.BooleanString().Optional()),
	zod.F("allow_backorder", commonvalidators.BooleanString().Optional()),
	zod.F("sku", utils.StringOrStrings().Optional()),
	zod.F("ean", utils.StringOrStrings().Optional()),
	zod.F("upc", utils.StringOrStrings().Optional()),
	zod.F("barcode", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetProductVariantsParams mirrors AdminGetProductVariantsParams.
var AdminGetProductVariantsParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(50)}).
	Merge(AdminGetProductVariantsParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetProductVariantsParamsFields))
