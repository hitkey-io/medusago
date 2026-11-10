// Package commonvalidators mirrors api/utils/common-validators.
package commonvalidators

import (
	"net/url"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
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

// RecursivelyNormalizeSchema mirrors recursivelyNormalizeSchema: the
// transform also applies to every $and / $or entry.
func RecursivelyNormalizeSchema(transform func(any) (any, error)) func(any) (any, error) {
	return func(v any) (any, error) {
		out, err := transform(v)
		if err != nil {
			return nil, err
		}
		r, ok := out.(*dal.Record)
		if !ok {
			return out, nil
		}
		for _, key := range r.Keys() {
			if key != "$and" && key != "$or" {
				continue
			}
			x, _ := r.Get(key)
			list, ok := x.([]any)
			if !ok {
				continue
			}
			mapped := make([]any, len(list))
			for i, item := range list {
				m, err := transform(item)
				if err != nil {
					return nil, err
				}
				mapped[i] = m
			}
			r.Set(key, mapped)
		}
		return r, nil
	}
}

// SafeHttpURL mirrors safeHttpUrl.
func SafeHttpURL() *zod.Schema {
	return zod.String().Refine(func(v any) bool {
		s, _ := v.(string)
		if s == "" || s == "#" {
			return true
		}
		u, err := url.Parse(s)
		if err != nil || u.Scheme == "" {
			return false
		}
		return u.Scheme == "http" || u.Scheme == "https"
	}, "URL must use http: or https: scheme")
}

// BigNumberInput mirrors BigNumberInput.
func BigNumberInput() *zod.Schema {
	return zod.Union(zod.Number(), zod.String(), zod.Object(
		zod.F("value", zod.String()),
		zod.F("precision", zod.Number()),
	))
}
