package utils

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// knexBuilder reproduces the subset of knex's query builder (pg dialect)
// the index query builder uses, compiling to the same SQL text: statements
// keep `?` placeholders and their bindings until toQuery() inlines them
// with the pg escaping (_formatQuery).

// knexRaw mirrors knex.raw(sql, bindings).
type knexRaw struct {
	sql      string
	bindings []any
}

type knexWhere struct {
	bool  string // "and" | "or"
	not   bool
	raw   *knexRaw
	group func(*knexBuilder) error
}

type knexColumn struct {
	expr     string // compiled column (identifier wrapped or raw)
	bindings []any
}

type knexBuilder struct {
	distinct bool
	columns  []knexColumn
	from     string
	joins    []knexRaw
	wheres   []knexWhere
	groups   []string
	orders   []knexRaw
	limit    *int
	offset   *int
}

func newKnexBuilder() *knexBuilder { return &knexBuilder{} }

var asRe = regexp.MustCompile(`(?i)\s+as\s+`)

// wrapIdentifier wraps "a.b" as "a"."b" and "t as a" as "t" as "a".
func wrapIdentifier(value string) string {
	if parts := asRe.Split(value, 2); len(parts) == 2 {
		return wrapIdentifier(parts[0]) + " as " + wrapSegment(parts[1])
	}
	segs := strings.Split(value, ".")
	for i, s := range segs {
		segs[i] = wrapSegment(s)
	}
	return strings.Join(segs, ".")
}

func wrapSegment(s string) string {
	s = strings.TrimSpace(s)
	if s == "*" {
		return s
	}
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func (b *knexBuilder) Distinct(col string) *knexBuilder {
	b.distinct = true
	b.columns = append(b.columns, knexColumn{expr: wrapIdentifier(col)})
	return b
}

func (b *knexBuilder) SelectRaw(raws ...knexRaw) *knexBuilder {
	for _, r := range raws {
		b.columns = append(b.columns, knexColumn{expr: r.sql, bindings: r.bindings})
	}
	return b
}

// selectPart is one entry of select({ alias: column }).
type selectPart struct {
	alias  string
	column string
	raw    *knexRaw
}

func (b *knexBuilder) SelectAliased(parts []selectPart) *knexBuilder {
	for _, p := range parts {
		if p.raw != nil {
			b.columns = append(b.columns, knexColumn{expr: p.raw.sql + " as " + wrapSegment(p.alias), bindings: p.raw.bindings})
			continue
		}
		b.columns = append(b.columns, knexColumn{expr: wrapIdentifier(p.column) + " as " + wrapSegment(p.alias)})
	}
	return b
}

func (b *knexBuilder) SelectLiteral(v string) *knexBuilder {
	b.columns = append(b.columns, knexColumn{expr: v})
	return b
}

func (b *knexBuilder) From(table string) *knexBuilder {
	b.from = wrapIdentifier(table)
	return b
}

func (b *knexBuilder) LeftJoin(table, first, second string) *knexBuilder {
	b.joins = append(b.joins, knexRaw{sql: "left join " + wrapIdentifier(table) + " on " + wrapIdentifier(first) + " = " + wrapIdentifier(second)})
	return b
}

func (b *knexBuilder) JoinRaw(sql string) *knexBuilder {
	b.joins = append(b.joins, knexRaw{sql: sql})
	return b
}

func (b *knexBuilder) WhereRaw(sql string, bindings ...any) *knexBuilder {
	b.wheres = append(b.wheres, knexWhere{bool: "and", raw: &knexRaw{sql: sql, bindings: bindings}})
	return b
}

func (b *knexBuilder) Where(fn func(*knexBuilder) error) *knexBuilder {
	b.wheres = append(b.wheres, knexWhere{bool: "and", group: fn})
	return b
}

func (b *knexBuilder) AndWhere(fn func(*knexBuilder) error) *knexBuilder { return b.Where(fn) }

func (b *knexBuilder) OrWhere(fn func(*knexBuilder) error) *knexBuilder {
	b.wheres = append(b.wheres, knexWhere{bool: "or", group: fn})
	return b
}

func (b *knexBuilder) WhereNot(fn func(*knexBuilder) error) *knexBuilder {
	b.wheres = append(b.wheres, knexWhere{bool: "and", not: true, group: fn})
	return b
}

func (b *knexBuilder) GroupBy(col string) *knexBuilder {
	b.groups = append(b.groups, wrapIdentifier(col))
	return b
}

func (b *knexBuilder) OrderByRaw(sql string) *knexBuilder {
	b.orders = append(b.orders, knexRaw{sql: sql})
	return b
}

func (b *knexBuilder) OrderBy(col, direction string) *knexBuilder {
	dir := direction
	if l := strings.ToLower(direction); l != "asc" && l != "desc" {
		dir = "asc"
	}
	b.orders = append(b.orders, knexRaw{sql: wrapIdentifier(col) + " " + dir})
	return b
}

func (b *knexBuilder) Limit(n int) *knexBuilder {
	b.limit = &n
	return b
}

func (b *knexBuilder) Offset(n int) *knexBuilder {
	b.offset = &n
	return b
}

// Clone mirrors clone().
func (b *knexBuilder) Clone() *knexBuilder {
	c := *b
	c.columns = append([]knexColumn(nil), b.columns...)
	c.joins = append([]knexRaw(nil), b.joins...)
	c.wheres = append([]knexWhere(nil), b.wheres...)
	c.groups = append([]string(nil), b.groups...)
	c.orders = append([]knexRaw(nil), b.orders...)
	return &c
}

// ClearSelect mirrors clearSelect().
func (b *knexBuilder) ClearSelect() *knexBuilder {
	b.distinct = false
	b.columns = nil
	return b
}

// ClearOrder mirrors clearOrder().
func (b *knexBuilder) ClearOrder() *knexBuilder {
	b.orders = nil
	return b
}

// compileRaw mirrors replaceRawArrBindings: every `?` takes the next
// binding (kept as a placeholder), `\?` stays.
func compileRaw(r knexRaw) (string, []any, error) {
	if len(r.bindings) == 0 {
		return r.sql, nil, nil
	}
	index := 0
	var out strings.Builder
	var bindings []any
	s := r.sql
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && s[i+1] == '?' {
			out.WriteString(`\?`)
			i++
			continue
		}
		if s[i] == '?' {
			if i+1 < len(s) && s[i+1] == '?' {
				// identifier binding (??)
				if index < len(r.bindings) {
					out.WriteString(wrapIdentifier(fmt.Sprint(r.bindings[index])))
				}
				index++
				i++
				continue
			}
			if index < len(r.bindings) {
				bindings = append(bindings, r.bindings[index])
			} else {
				bindings = append(bindings, nil)
			}
			index++
			out.WriteByte('?')
			continue
		}
		out.WriteByte(s[i])
	}
	if index != len(r.bindings) {
		return "", nil, fmt.Errorf("Expected %d bindings, saw %d", len(r.bindings), index)
	}
	return out.String(), bindings, nil
}

// compileWhere compiles the where component ("where ...").
func (b *knexBuilder) compileWhere() (string, []any, error) {
	var sql []string
	var bindings []any
	for _, w := range b.wheres {
		var val string
		if w.raw != nil {
			s, bs, err := compileRaw(*w.raw)
			if err != nil {
				return "", nil, err
			}
			val = s
			bindings = append(bindings, bs...)
			if w.not {
				val = "not " + val
			}
		} else {
			sub := newKnexBuilder()
			if err := w.group(sub); err != nil {
				return "", nil, err
			}
			s, bs, err := sub.compileWhere()
			if err != nil {
				return "", nil, err
			}
			if s != "" {
				val = "(" + s[len("where "):] + ")"
				if w.not {
					val = "not " + val
				}
				bindings = append(bindings, bs...)
			}
		}
		if val == "" {
			continue
		}
		if len(sql) == 0 {
			sql = append(sql, "where")
		} else {
			sql = append(sql, w.bool)
		}
		sql = append(sql, val)
	}
	if len(sql) > 1 {
		return strings.Join(sql, " "), bindings, nil
	}
	return "", bindings, nil
}

// toSQL compiles the statement with `?` placeholders.
func (b *knexBuilder) toSQL() (string, []any, error) {
	var parts []string
	var bindings []any
	cols := make([]string, 0, len(b.columns))
	for _, c := range b.columns {
		cols = append(cols, c.expr)
		bindings = append(bindings, c.bindings...)
	}
	if len(cols) == 0 {
		cols = []string{"*"}
	}
	sel := "select "
	if b.distinct {
		sel += "distinct "
	}
	sel += strings.Join(cols, ", ")
	if b.from != "" {
		sel += " from " + b.from
	}
	parts = append(parts, sel)
	for _, j := range b.joins {
		s, bs, err := compileRaw(j)
		if err != nil {
			return "", nil, err
		}
		parts = append(parts, s)
		bindings = append(bindings, bs...)
	}
	where, wb, err := b.compileWhere()
	if err != nil {
		return "", nil, err
	}
	if where != "" {
		parts = append(parts, where)
		bindings = append(bindings, wb...)
	}
	if len(b.groups) > 0 {
		parts = append(parts, "group by "+strings.Join(b.groups, ", "))
	}
	if len(b.orders) > 0 {
		os := make([]string, 0, len(b.orders))
		for _, o := range b.orders {
			s, bs, err := compileRaw(o)
			if err != nil {
				return "", nil, err
			}
			os = append(os, s)
			bindings = append(bindings, bs...)
		}
		parts = append(parts, "order by "+strings.Join(os, ", "))
	}
	if b.limit != nil {
		parts = append(parts, "limit ?")
		bindings = append(bindings, float64(*b.limit))
	}
	if b.offset != nil {
		parts = append(parts, "offset ?")
		bindings = append(bindings, float64(*b.offset))
	}
	return strings.Join(parts, " "), bindings, nil
}

// ToQuery mirrors toQuery(): the SQL with the bindings inlined.
func (b *knexBuilder) ToQuery() (string, error) {
	sql, bindings, err := b.toSQL()
	if err != nil {
		return "", err
	}
	return formatQuery(sql, bindings), nil
}

// rawToQuery mirrors knex.raw(sql, bindings).toQuery().
func rawToQuery(sql string, bindings ...any) (string, error) {
	s, bs, err := compileRaw(knexRaw{sql: sql, bindings: bindings})
	if err != nil {
		return "", err
	}
	return formatQuery(s, bs), nil
}

// formatQuery mirrors client._formatQuery: each `?` takes the next
// escaped binding, `\?` becomes `?`.
func formatQuery(sql string, bindings []any) string {
	var out strings.Builder
	index := 0
	for i := 0; i < len(sql); i++ {
		if sql[i] == '\\' && i+1 < len(sql) && sql[i+1] == '?' {
			out.WriteByte('?')
			i++
			continue
		}
		if sql[i] == '?' {
			if index == len(bindings) {
				out.WriteByte('?')
				continue
			}
			out.WriteString(escapeBinding(bindings[index]))
			index++
			continue
		}
		out.WriteByte(sql[i])
	}
	return out.String()
}

// escapeBinding mirrors the pg client's _escapeBinding.
func escapeBinding(v any) string {
	switch t := v.(type) {
	case nil:
		return "NULL"
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return jsNumberString(t)
	case int:
		return strconv.Itoa(t)
	case string:
		return pgEscapeString(t)
	case *dal.Record:
		b, _ := t.MarshalJSON()
		return string(b)
	default:
		return pgEscapeString(fmt.Sprint(t))
	}
}

func pgEscapeString(str string) string {
	hasBackslash := false
	var b strings.Builder
	b.WriteByte('\'')
	for _, c := range str {
		switch c {
		case '\'':
			b.WriteString("''")
		case '\\':
			b.WriteString(`\\`)
			hasBackslash = true
		default:
			b.WriteRune(c)
		}
	}
	b.WriteByte('\'')
	if hasBackslash {
		return "E" + b.String()
	}
	return b.String()
}

// jsNumberString formats a number like Number.prototype.toString().
func jsNumberString(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	case f == 0:
		return "0"
	}
	abs := math.Abs(f)
	if abs >= 1e21 || abs < 1e-6 {
		s := strconv.FormatFloat(f, 'e', -1, 64)
		mant, exp, _ := strings.Cut(s, "e")
		sign := exp[0]
		exp = strings.TrimLeft(exp[1:], "0")
		if exp == "" {
			exp = "0"
		}
		return mant + "e" + string(sign) + exp
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}
