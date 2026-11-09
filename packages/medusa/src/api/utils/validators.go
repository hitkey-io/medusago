// Package utils mirrors @medusajs/medusa/src/api/utils.
package utils

import (
	"math"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
)

// ParseInt mirrors JavaScript's parseInt(string).
func ParseInt(s string) float64 {
	s = strings.TrimLeft(s, " \t\n\r\v\f\u00a0\u2028\u2029"+string(rune(0xFEFF)))
	sign := 1.0
	if strings.HasPrefix(s, "-") {
		sign = -1
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	base := 10.0
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		base = 16
		s = s[2:]
	}
	n := 0.0
	digits := 0
	for _, c := range s {
		var d float64
		switch {
		case c >= '0' && c <= '9':
			d = float64(c - '0')
		case base == 16 && c >= 'a' && c <= 'f':
			d = float64(c-'a') + 10
		case base == 16 && c >= 'A' && c <= 'F':
			d = float64(c-'A') + 10
		default:
			goto done
		}
		n = n*base + d
		digits++
	}
done:
	if digits == 0 {
		return math.NaN()
	}
	return sign * n
}

func toNumberPreprocess(v any) any {
	if s, ok := v.(string); ok && s != "" {
		return ParseInt(s)
	}
	return v
}

// CreateSelectParams mirrors createSelectParams.
func CreateSelectParams() *zod.Schema {
	return zod.Object(zod.F("fields", zod.String().Optional()))
}

// FindParamsOptions mirrors the options of createFindParams.
type FindParamsOptions struct {
	Offset *float64
	Limit  *float64
	Order  string
}

// CreateFindParams mirrors createFindParams.
func CreateFindParams(o FindParamsOptions) *zod.Schema {
	offset, limit := 0.0, 20.0
	if o.Offset != nil {
		offset = *o.Offset
	}
	if o.Limit != nil {
		limit = *o.Limit
	}
	order := zod.String().Optional()
	if o.Order != "" {
		order = zod.String().Optional().Default(o.Order)
	}
	return CreateSelectParams().Merge(zod.Object(
		zod.F("offset", zod.Preprocess(toNumberPreprocess, zod.Number().Optional().Default(offset))),
		zod.F("limit", zod.Preprocess(toNumberPreprocess, zod.Number().Optional().Default(limit))),
		zod.F("order", order),
		zod.F("with_deleted", zod.Preprocess(func(v any) any {
			if s, ok := v.(string); ok && s != "" {
				switch s {
				case "true":
					return true
				case "false":
					return false
				}
			}
			return v
		}, zod.Boolean().Optional())),
	))
}

// Num is a helper for FindParamsOptions.
func Num(f float64) *float64 { return &f }

// CreateOperatorMap mirrors createOperatorMap (type nil = z.string()).
func CreateOperatorMap(typ *zod.Schema, valueParser func(any) any) *zod.Schema {
	if typ == nil {
		typ = zod.String()
	}
	simple := typ.Optional()
	if valueParser != nil {
		simple = zod.Preprocess(valueParser, typ).Optional()
	}
	arrayType := zod.Array(typ).Optional()
	unionType := zod.Union(simple, arrayType).Optional()
	return zod.Union(
		unionType,
		zod.Object(
			zod.F("$eq", unionType),
			zod.F("$ne", unionType),
			zod.F("$in", arrayType),
			zod.F("$nin", arrayType),
			zod.F("$like", simple),
			zod.F("$ilike", simple),
			zod.F("$re", simple),
			zod.F("$contains", simple),
			zod.F("$gt", simple),
			zod.F("$gte", simple),
			zod.F("$lt", simple),
			zod.F("$lte", simple),
		),
	)
}

// CreateBatchBody mirrors createBatchBody.
func CreateBatchBody(create, update, del *zod.Schema) *zod.Schema {
	if del == nil {
		del = zod.String()
	}
	return zod.Object(
		zod.F("create", zod.Array(create).Optional()),
		zod.F("update", zod.Array(update).Optional()),
		zod.F("delete", zod.Array(del).Optional()),
	)
}

// CreateLinkBody mirrors createLinkBody.
func CreateLinkBody() *zod.Schema {
	return zod.Object(
		zod.F("add", zod.Array(zod.String()).Optional()),
		zod.F("remove", zod.Array(zod.String()).Optional()),
	)
}

// StringOrStrings is z.union([z.string(), z.array(z.string())]).
func StringOrStrings() *zod.Schema {
	return zod.Union(zod.String(), zod.Array(zod.String()))
}
