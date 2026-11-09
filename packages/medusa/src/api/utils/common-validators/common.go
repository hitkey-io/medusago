// Package commonvalidators mirrors api/utils/common-validators.
package commonvalidators

import (
	"strings"

	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
)

// ApplyAndAndOrOperators mirrors applyAndAndOrOperators.
func ApplyAndAndOrOperators(schema *zod.Schema) *zod.Schema {
	return schema.Merge(zod.Object(
		zod.F("$and", zod.Lazy(func() *zod.Schema { return schema.Array() }).Optional()),
		zod.F("$or", zod.Lazy(func() *zod.Schema { return schema.Array() }).Optional()),
	))
}

// BooleanString mirrors booleanString.
func BooleanString() *zod.Schema {
	return zod.Union(zod.Boolean(), zod.String()).Refine(func(v any) bool {
		s := strings.ToLower(zod.JSString(v))
		return s == "true" || s == "false"
	}, "").Transform(func(v any) (any, error) {
		return strings.ToLower(zod.JSString(v)) == "true", nil
	})
}

// AddressPayload mirrors AddressPayload.
func AddressPayload() *zod.Schema {
	return zod.Object(
		zod.F("first_name", zod.String().Nullish()),
		zod.F("last_name", zod.String().Nullish()),
		zod.F("phone", zod.String().Nullish()),
		zod.F("company", zod.String().Nullish()),
		zod.F("address_1", zod.String().Nullish()),
		zod.F("address_2", zod.String().Nullish()),
		zod.F("city", zod.String().Nullish()),
		zod.F("country_code", zod.String().Nullish()),
		zod.F("province", zod.String().Nullish()),
		zod.F("postal_code", zod.String().Nullish()),
		zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
	).Strict()
}
