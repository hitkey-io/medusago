package propertylabels

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// AdminPropertyLabelParams mirrors AdminPropertyLabelParams.
var AdminPropertyLabelParams = utils.CreateSelectParams()

// AdminPropertyLabelListParams mirrors AdminPropertyLabelListParams.
var AdminPropertyLabelListParams = utils.CreateSelectParams().Extend(
	zod.F("entity", zod.String().Optional()),
	zod.F("property", zod.String().Optional()),
	zod.F("q", zod.String().Optional()),
	zod.F("limit", zod.CoerceNumber().Optional()),
	zod.F("offset", zod.CoerceNumber().Optional()),
	zod.F("order", zod.String().Optional()),
)

// AdminCreatePropertyLabel mirrors AdminCreatePropertyLabel.
var AdminCreatePropertyLabel = zod.Object(
	zod.F("entity", zod.String().Min(1)),
	zod.F("property", zod.String().Min(1)),
	zod.F("label", zod.String().Min(1)),
	zod.F("description", zod.String().Nullish()),
)

// AdminUpdatePropertyLabel mirrors AdminUpdatePropertyLabel.
var AdminUpdatePropertyLabel = zod.Object(
	zod.F("label", zod.String().Min(1).Optional()),
	zod.F("description", zod.String().Nullish()),
)

// AdminBatchPropertyLabels mirrors AdminBatchPropertyLabels.
var AdminBatchPropertyLabels = zod.Object(
	zod.F("create", zod.Array(zod.Object(
		zod.F("entity", zod.String().Min(1)),
		zod.F("property", zod.String().Min(1)),
		zod.F("label", zod.String().Min(1)),
		zod.F("description", zod.String().Nullish()),
	)).Optional()),
	zod.F("update", zod.Array(zod.Object(
		zod.F("id", zod.String().Min(1)),
		zod.F("label", zod.String().Min(1).Optional()),
		zod.F("description", zod.String().Nullish()),
	)).Optional()),
	zod.F("delete", zod.Array(zod.String()).Optional()),
)
