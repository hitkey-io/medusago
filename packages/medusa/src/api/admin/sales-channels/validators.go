package saleschannels

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// AdminGetSalesChannelParams mirrors AdminGetSalesChannelParams.
var AdminGetSalesChannelParams = utils.CreateSelectParams()

// AdminGetSalesChannelsParamsDirectFields mirrors
// AdminGetSalesChannelsParamsDirectFields.
var AdminGetSalesChannelsParamsDirectFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("name", utils.StringOrStrings().Optional()),
	zod.F("description", zod.String().Optional()),
	zod.F("is_disabled", commonvalidators.BooleanString().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetSalesChannelsParams mirrors AdminGetSalesChannelsParams.
var AdminGetSalesChannelsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(20), Offset: utils.Num(0)}).
	Merge(AdminGetSalesChannelsParamsDirectFields).
	Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetSalesChannelsParamsDirectFields)).
	Merge(zod.Object(
		zod.F("location_id", utils.StringOrStrings().Optional()),
		zod.F("publishable_key_id", utils.StringOrStrings().Optional()),
	))
