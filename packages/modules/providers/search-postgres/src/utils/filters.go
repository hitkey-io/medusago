package utils

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	searchutils "github.com/hitkey-io/medusago/packages/core/utils/src/search"
)

// SqlFragment mirrors SqlFragment: SQL with `?` placeholders and their
// values.
type SqlFragment struct {
	SQL    string
	Params []any
}

// JsonbPath mirrors jsonbPath.
func JsonbPath(path string, asText bool) string {
	segments := strings.Split(path, ".")
	expr := "indexed"
	for i, s := range segments {
		segment := strings.ReplaceAll(s, "'", "''")
		if i == len(segments)-1 && asText {
			expr += "->>'" + segment + "'"
		} else {
			expr += "->'" + segment + "'"
		}
	}
	return expr
}

// castScalar mirrors castScalar (nil means null).
func castScalar(planned *PlannedField, value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	switch planned.Kind {
	case "number":
		var num float64
		if t, ok := value.(time.Time); ok && planned.IsDate {
			num = float64(t.UnixMilli())
		} else {
			num = searchutils.JSNumber(value)
		}
		if math.IsNaN(num) {
			return nil, fail(fmt.Sprintf(`Cannot cast filter value for "%s" to a number`, planned.Path))
		}
		return num, nil
	case "boolean":
		return searchutils.SearchValueToBoolean(value), nil
	}
	if s, ok := value.(string); ok {
		return s, nil
	}
	return searchutils.JSString(value), nil
}

func nextParam(params *[]any, value any) string {
	*params = append(*params, value)
	return "?"
}

// containmentSql mirrors containmentSql: `"indexed" @> '{"a":{"b":<leaf>}}'`.
func containmentSql(path string, leaf any, params *[]any) (string, error) {
	segments := strings.Split(path, ".")
	doc := leaf
	for i := len(segments) - 1; i >= 0; i-- {
		r := dal.NewRecord()
		r.Set(segments[i], doc)
		doc = r
	}
	raw, err := dal.MarshalValue(doc)
	if err != nil {
		return "", err
	}
	return `"indexed" @> ` + nextParam(params, string(raw)) + "::jsonb", nil
}

func existsSql(planned *PlannedField, exists bool) string {
	expr := JsonbPath(planned.Path, false)
	if exists {
		return fmt.Sprintf("(%s IS NOT NULL AND %s <> 'null'::jsonb)", expr, expr)
	}
	return fmt.Sprintf("(%s IS NULL OR %s = 'null'::jsonb)", expr, expr)
}

func equalitySql(planned *PlannedField, value any, params *[]any) (string, error) {
	cast, err := castScalar(planned, value)
	if err != nil {
		return "", err
	}
	if cast == nil {
		return existsSql(planned, false), nil
	}
	var leaf any = cast
	if planned.IsArray {
		leaf = []any{cast}
	}
	return containmentSql(planned.Path, leaf, params)
}

func anyOfSql(planned *PlannedField, values []any, params *[]any) (string, error) {
	if len(values) == 0 {
		return "FALSE", nil
	}
	parts := make([]string, 0, len(values))
	for _, v := range values {
		p, err := equalitySql(planned, v, params)
		if err != nil {
			return "", err
		}
		parts = append(parts, p)
	}
	if len(parts) == 1 {
		return parts[0], nil
	}
	return "(" + strings.Join(parts, " OR ") + ")", nil
}

func comparisonSql(planned *PlannedField, operator string, value any, params *[]any) (string, error) {
	if planned.IsArray {
		return "", fail(fmt.Sprintf(`Operator "%s" is not supported on array field "%s"`, operator, planned.Path))
	}
	cast, err := castScalar(planned, value)
	if err != nil {
		return "", err
	}
	switch planned.Kind {
	case "number":
		return fmt.Sprintf("(%s)::float8 %s %s", JsonbPath(planned.Path, true), operator, nextParam(params, cast)), nil
	case "boolean":
		return fmt.Sprintf("(%s)::boolean %s %s", JsonbPath(planned.Path, true), operator, nextParam(params, cast)), nil
	}
	return fmt.Sprintf("%s %s %s", JsonbPath(planned.Path, true), operator, nextParam(params, cast)), nil
}

func arrayContainsSql(planned *PlannedField, values []any, mode string, params *[]any) (string, error) {
	if !planned.IsArray {
		op := "overlaps"
		if mode == "all" {
			op = "contains"
		}
		return "", fail(fmt.Sprintf(`Operator $%s on "%s" needs an array field`, op, planned.Path))
	}
	if mode == "any" {
		return anyOfSql(planned, values, params)
	}
	cast := make([]any, len(values))
	for i, v := range values {
		c, err := castScalar(planned, v)
		if err != nil {
			return "", err
		}
		cast[i] = c
	}
	return containmentSql(planned.Path, cast, params)
}

func listOf(v any) []any {
	if arr, ok := v.([]any); ok {
		return arr
	}
	return []any{v}
}

func fieldPredicate(path string, raw any, plan *IndexPlan, params *[]any) (string, error) {
	planned := plan.Field(path)
	if planned == nil {
		return "", fail(fmt.Sprintf(`Unknown filter field "%s" on the postgres search provider`, path))
	}
	operators, isRecord := raw.(*dal.Record)
	if !isRecord {
		return anyOfSql(planned, listOf(raw), params)
	}
	var keys []string
	for _, k := range operators.Keys() {
		if strings.HasPrefix(k, "$") {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return "", fail(fmt.Sprintf(`Empty operator map on filter field "%s"`, path))
	}
	parts := []string{}
	for _, operator := range keys {
		value, _ := operators.Get(operator)
		var part string
		var err error
		switch operator {
		case "$eq":
			part, err = equalitySql(planned, value, params)
		case "$ne":
			if value == nil {
				part = existsSql(planned, true)
			} else {
				part, err = equalitySql(planned, value, params)
				part = "NOT " + part
			}
		case "$in":
			part, err = anyOfSql(planned, listOf(value), params)
		case "$nin":
			part, err = anyOfSql(planned, listOf(value), params)
			part = "NOT " + part
		case "$gt":
			part, err = comparisonSql(planned, ">", value, params)
		case "$gte":
			part, err = comparisonSql(planned, ">=", value, params)
		case "$lt":
			part, err = comparisonSql(planned, "<", value, params)
		case "$lte":
			part, err = comparisonSql(planned, "<=", value, params)
		case "$exists":
			part = existsSql(planned, searchutils.Truthy(value))
		case "$contains":
			part, err = arrayContainsSql(planned, listOf(value), "all", params)
		case "$overlaps":
			part, err = arrayContainsSql(planned, listOf(value), "any", params)
		case "$prefix", "$like":
			if planned.IsArray {
				return "", fail(fmt.Sprintf(`Operator "%s" is not supported on array field "%s"`, operator, path))
			}
			pattern := searchutils.JSString(value)
			if operator == "$prefix" {
				pattern += "%"
			}
			part = fmt.Sprintf("%s LIKE %s", JsonbPath(planned.Path, true), nextParam(params, pattern))
		default:
			return "", fail(fmt.Sprintf(`The postgres search provider does not support operator "%s" on "%s"`, operator, path))
		}
		if err != nil {
			return "", err
		}
		parts = append(parts, part)
	}
	if len(parts) == 1 {
		return parts[0], nil
	}
	return "(" + strings.Join(parts, " AND ") + ")", nil
}

// ToWhereClause mirrors toWhereClause: the filter tree as a SQL boolean
// expression (nil when it compiles to nothing).
func ToWhereClause(filters *dal.Record, plan *IndexPlan) (*SqlFragment, error) {
	if filters == nil {
		return nil, nil
	}
	params := []any{}
	var compile func(branch *dal.Record) (string, error)
	compileList := func(value any) ([]string, error) {
		arr, _ := value.([]any)
		var nested []string
		for _, b := range arr {
			br, _ := b.(*dal.Record)
			sql, err := compile(br)
			if err != nil {
				return nil, err
			}
			if sql != "" {
				nested = append(nested, sql)
			}
		}
		return nested, nil
	}
	compile = func(branch *dal.Record) (string, error) {
		if branch == nil {
			return "", nil
		}
		parts := []string{}
		for _, key := range branch.Keys() {
			value, _ := branch.Get(key)
			switch key {
			case "q":
				continue
			case "$and", "$or":
				nested, err := compileList(value)
				if err != nil {
					return "", err
				}
				if len(nested) > 0 {
					sep := " AND "
					if key == "$or" {
						sep = " OR "
					}
					parts = append(parts, "("+strings.Join(nested, sep)+")")
				}
				continue
			case "$not":
				nb, _ := value.(*dal.Record)
				nested, err := compile(nb)
				if err != nil {
					return "", err
				}
				if nested != "" {
					parts = append(parts, "NOT ("+nested+")")
				}
				continue
			}
			p, err := fieldPredicate(key, value, plan, &params)
			if err != nil {
				return "", err
			}
			parts = append(parts, p)
		}
		return strings.Join(parts, " AND "), nil
	}
	sql, err := compile(filters)
	if err != nil {
		return nil, err
	}
	if sql == "" {
		return nil, nil
	}
	return &SqlFragment{SQL: sql, Params: params}, nil
}
