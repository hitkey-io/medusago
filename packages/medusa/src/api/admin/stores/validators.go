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
