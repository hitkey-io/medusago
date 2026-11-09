package configurations

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// AdminGetLayoutConfigurationsParamsFields mirrors AdminGetLayoutConfigurationsParamsFields.
var AdminGetLayoutConfigurationsParamsFields = zod.Object(
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("zone", utils.StringOrStrings().Optional()),
	zod.F("user_id", zod.Union(zod.String(), zod.Array(zod.String()), zod.Null()).Optional()),
	zod.F("is_system_default", zod.Boolean().Optional()),
)

// AdminGetLayoutConfigurationsParams mirrors AdminGetLayoutConfigurationsParams.
var AdminGetLayoutConfigurationsParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(20)}).
	Merge(AdminGetLayoutConfigurationsParamsFields).
	Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetLayoutConfigurationsParamsFields))
