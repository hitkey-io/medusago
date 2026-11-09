package collections

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// CreateCollection mirrors CreateCollection.
var CreateCollection = zod.Object(
	zod.F("title", zod.String()),
	zod.F("handle", zod.String().Optional()),
	zod.F("external_id", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

// AdminCreateCollection mirrors AdminCreateCollection.
var AdminCreateCollection = utils.WithAdditionalData(CreateCollection)

// UpdateCollection mirrors UpdateCollection.
var UpdateCollection = zod.Object(
	zod.F("title", zod.String().Optional()),
	zod.F("handle", zod.String().Optional()),
	zod.F("external_id", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

// AdminUpdateCollection mirrors AdminUpdateCollection.
var AdminUpdateCollection = utils.WithAdditionalData(UpdateCollection)
