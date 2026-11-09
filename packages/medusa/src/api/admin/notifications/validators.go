package notifications

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// AdminGetNotificationParams mirrors AdminGetNotificationParams.
var AdminGetNotificationParams = utils.CreateSelectParams()

// AdminGetNotificationsParamsFields mirrors AdminGetNotificationsParamsFields.
var AdminGetNotificationsParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("channel", utils.StringOrStrings().Optional()),
	zod.F("to", utils.StringOrStrings().Optional()),
)

// AdminGetNotificationsParams mirrors AdminGetNotificationsParams.
var AdminGetNotificationsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(50), Offset: utils.Num(0), Order: "-created_at"}).
	Merge(AdminGetNotificationsParamsFields).
	Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetNotificationsParamsFields))
