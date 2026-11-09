package regions

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// AdminGetRegionParams mirrors AdminGetRegionParams.
var AdminGetRegionParams = utils.CreateSelectParams()

// AdminGetRegionsParamsFields mirrors AdminGetRegionsParamsFields.
var AdminGetRegionsParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("currency_code", utils.StringOrStrings().Optional()),
	zod.F("name", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetRegionsParams mirrors AdminGetRegionsParams.
var AdminGetRegionsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(50), Offset: utils.Num(0)}).
	Merge(AdminGetRegionsParamsFields).
	Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetRegionsParamsFields))

// AdminCreateRegion mirrors AdminCreateRegion.
var AdminCreateRegion = zod.Object(
	zod.F("name", zod.String()),
	zod.F("currency_code", zod.String()),
	zod.F("countries", zod.Array(zod.String()).Optional()),
	zod.F("automatic_taxes", zod.Boolean().Optional()),
	zod.F("is_tax_inclusive", zod.Boolean().Optional()),
	zod.F("payment_providers", zod.Array(zod.String()).Optional()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
).Strict()

// AdminUpdateRegion mirrors AdminUpdateRegion.
var AdminUpdateRegion = zod.Object(
	zod.F("name", zod.String().Optional()),
	zod.F("currency_code", zod.String().Optional()),
	zod.F("countries", zod.Array(zod.String()).Optional()),
	zod.F("automatic_taxes", zod.Boolean().Optional()),
	zod.F("is_tax_inclusive", zod.Boolean().Optional()),
	zod.F("payment_providers", zod.Array(zod.String()).Optional()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
).Strict()
