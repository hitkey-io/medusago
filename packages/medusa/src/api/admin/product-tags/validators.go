package producttags

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
)

// AdminCreateProductTag mirrors AdminCreateProductTag.
var AdminCreateProductTag = zod.Object(
	zod.F("value", zod.String()),
	zod.F("external_id", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
).Strict()

// AdminUpdateProductTag mirrors AdminUpdateProductTag.
var AdminUpdateProductTag = zod.Object(
	zod.F("value", zod.String().Optional()),
	zod.F("external_id", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
).Strict()
