package users

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// AdminGetUserParams mirrors AdminGetUserParams.
var AdminGetUserParams = utils.CreateSelectParams()

// AdminGetUsersParams mirrors AdminGetUsersParams.
var AdminGetUsersParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(50)}).Merge(zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("email", zod.String().Nullish()),
	zod.F("first_name", zod.String().Nullish()),
	zod.F("last_name", zod.String().Nullish()),
))
