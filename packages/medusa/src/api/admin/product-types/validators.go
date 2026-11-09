package producttypes

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
)

// AdminCreateProductType mirrors AdminCreateProductType.
var AdminCreateProductType = zod.Object(
	zod.F("value", zod.String()),
	zod.F("external_id", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
).Strict()

// AdminUpdateProductType mirrors AdminUpdateProductType.
var AdminUpdateProductType = zod.Object(
	zod.F("value", zod.String().Optional()),
	zod.F("external_id", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
).Strict()
