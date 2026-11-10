// Package orderchanges mirrors api/admin/order-changes.
package orderchanges

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// AdminOrderChangeParams mirrors AdminOrderChangeParams.
var AdminOrderChangeParams = utils.CreateSelectParams().Merge(zod.Object(
	zod.F("id", zod.Union(zod.String(), zod.Array(zod.String())).Optional()),
	zod.F("status", zod.Union(zod.String(), zod.Array(zod.String())).Optional()),
	zod.F("change_type", zod.Union(zod.String(), zod.Array(zod.String())).Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
))

// AdminPostOrderChangesReqSchema mirrors AdminPostOrderChangesReqSchema.
var AdminPostOrderChangesReqSchema = zod.Object(
	zod.F("carry_over_promotions", zod.Boolean().Nullish()),
	zod.F("internal_note", zod.String().Nullish()),
)
