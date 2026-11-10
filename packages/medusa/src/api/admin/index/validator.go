package index

import "github.com/hitkey-io/medusago/packages/core/framework/src/zod"

// AdminIndexSyncPayload mirrors AdminIndexSyncPayload.
var AdminIndexSyncPayload = zod.Object(
	zod.F("strategy", zod.Enum("full", "reset").Optional()),
)
