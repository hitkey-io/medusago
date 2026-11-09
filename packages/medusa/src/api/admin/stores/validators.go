package stores

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// AdminGetStoreParams mirrors AdminGetStoreParams.
var AdminGetStoreParams = utils.CreateSelectParams()

// AdminGetStoresParamsFields mirrors AdminGetStoresParamsFields.
var AdminGetStoresParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("name", utils.StringOrStrings().Optional()),
)

// AdminGetStoresParams mirrors AdminGetStoresParams.
var AdminGetStoresParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(50), Offset: utils.Num(0)}).
	Merge(AdminGetStoresParamsFields).
	Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetStoresParamsFields))

// AdminUpdateStore mirrors AdminUpdateStore.
var AdminUpdateStore = zod.Object(
	zod.F("name", zod.String().Optional()),
	zod.F("supported_currencies", zod.Array(zod.Object(
		zod.F("currency_code", zod.String()),
		zod.F("is_default", zod.Boolean().Optional()),
		zod.F("is_tax_inclusive", zod.Boolean().Optional()),
	)).Optional()),
	zod.F("supported_locales", zod.Array(zod.Object(
		zod.F("locale_code", zod.String()),
	)).Optional()),
	zod.F("default_sales_channel_id", zod.String().Nullish()),
	zod.F("default_region_id", zod.String().Nullish()),
	zod.F("default_location_id", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)
