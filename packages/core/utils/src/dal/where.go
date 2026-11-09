package dal

import (
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
)

// queryBuilder compiles MikroORM style filters to SQL.
type queryBuilder struct {
	schema      *Schema
	withDeleted bool
	args        []any
	joins       []string
	joinAliases map[string]string
	aliasN      int
	toManyJoins bool
}

func newQueryBuilder(schema *Schema, withDeleted bool) *queryBuilder {
	return &queryBuilder{schema: schema, withDeleted: withDeleted, joinAliases: map[string]string{}}
}

func (q *queryBuilder) arg(v any) string {
	q.args = append(q.args, normalizeArg(v))
	return fmt.Sprintf("$%d", len(q.args))
}

// normalizeArg turns JSON numbers that are integers into int64 so they bind
// to integer columns.
func normalizeArg(v any) any {
	if f, ok := v.(float64); ok && f == float64(int64(f)) {
		return int64(f)
	}
	return v
}

func (q *queryBuilder) nextAlias() string {
	q.aliasN++
	return fmt.Sprintf("j%d", q.aliasN)
}

func (q *queryBuilder) joinSQL() string {
	if len(q.joins) == 0 {
		return ""
	}
	return " " + strings.Join(q.joins, " ")
}

// rootWhere returns the WHERE clause (with leading " where ") for the root
// entity, including the soft delete filter.
func (q *queryBuilder) rootWhere(e *dml.Entity, alias string, filter *Record) (string, error) {
	var conds []string
	if !q.withDeleted && hasDeletedAt(e) {
		conds = append(conds, col(alias, "deleted_at")+" is null")
	}
	if filter != nil {
		c, err := q.where(e, alias, filter, alias)
		if err != nil {
			return "", err
		}
		if c != "" {
			conds = append(conds, c)
		}
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " where " + strings.Join(conds, " and "), nil
}

func isOperatorObject(r *Record) bool {
	if r == nil || r.Len() == 0 {
		return false
	}
	for _, k := range r.Keys() {
		if !strings.HasPrefix(k, "$") {
			return false
		}
	}
	return true
}

// where compiles a filter object for the entity at alias. path identifies
// the alias chain used for relation joins.
func (q *queryBuilder) where(e *dml.Entity, alias string, filter *Record, path string) (string, error) {
	var conds []string
	for _, key := range filter.Keys() {
		value, _ := filter.Get(key)
		var c string
		var err error
		switch key {
		case "$and", "$or":
			c, err = q.logical(e, alias, key, value, path)
		case "$not":
			sub, ok := value.(*Record)
			if !ok {
				return "", fmt.Errorf("$not expects an object")
			}
			c, err = q.where(e, alias, sub, path)
			if c != "" {
				c = "not (" + c + ")"
			}
		case "q":
			c, err = q.freeText(e, alias, fmt.Sprint(value), path)
		default:
			c, err = q.property(e, alias, key, value, path)
		}
		if err != nil {
			return "", err
		}
		if c != "" {
			conds = append(conds, c)
		}
	}
	return strings.Join(conds, " and "), nil
}

func (q *queryBuilder) logical(e *dml.Entity, alias, op string, value any, path string) (string, error) {
	items, ok := value.([]any)
	if !ok {
		return "", fmt.Errorf("Expected array for %s but got %v", op, value)
	}
	var parts []string
	for _, item := range items {
		sub, ok := item.(*Record)
		if !ok {
			return "", fmt.Errorf("%s expects objects", op)
		}
		c, err := q.where(e, alias, sub, path)
		if err != nil {
			return "", err
		}
		if c != "" {
			parts = append(parts, "("+c+")")
		}
	}
	if len(parts) == 0 {
		return "", nil
	}
	sep := " and "
	if op == "$or" {
		sep = " or "
	}
	return "(" + strings.Join(parts, sep) + ")", nil
}

func (q *queryBuilder) property(e *dml.Entity, alias, key string, value any, path string) (string, error) {
	prop, ok := e.MetaProperty(key)
	if !ok {
		return "", fmt.Errorf("%s has no property %s", e.Meta.ClassName, key)
	}

	isRelation := prop.Kind != "scalar" && !(prop.MapToPk && (prop.Persist == nil || *prop.Persist))
	obj, isObj := value.(*Record)

	if isRelation && isObj && !isOperatorObject(obj) {
		targetAlias, target, err := q.joinRelation(e, alias, key, path)
		if err != nil {
			return "", err
		}
		return q.where(target, targetAlias, obj, path+"."+key)
	}

	column := ""
	if len(prop.FieldNames) > 0 {
		column = col(alias, prop.FieldNames[0])
	}
	if column == "" {
		return "", fmt.Errorf("%s.%s cannot be filtered", e.Meta.ClassName, key)
	}

	if isObj && !isOperatorObject(obj) && prop.ColumnType == "jsonb" {
		return q.jsonFilter(column, obj)
	}
	return q.operators(column, value)
}

func (q *queryBuilder) jsonFilter(column string, obj *Record) (string, error) {
	var conds []string
	for _, k := range obj.Keys() {
		v, _ := obj.Get(k)
		c, err := q.operators(fmt.Sprintf("%s->>'%s'", column, strings.ReplaceAll(k, "'", "''")), v)
		if err != nil {
			return "", err
		}
		conds = append(conds, c)
	}
	return strings.Join(conds, " and "), nil
}

func (q *queryBuilder) inList(column string, values []any, negate bool) string {
	if len(values) == 0 {
		if negate {
			return "1 = 1"
		}
		return "1 = 0"
	}
	ph := make([]string, len(values))
	for i, v := range values {
		ph[i] = q.arg(v)
	}
	op := " in "
	if negate {
		op = " not in "
	}
	return column + op + "(" + strings.Join(ph, ", ") + ")"
}

func (q *queryBuilder) operators(column string, value any) (string, error) {
	switch v := value.(type) {
	case nil:
		return column + " is null", nil
	case []any:
		return q.inList(column, dedupe(v), false), nil
	case *Record:
		var conds []string
		for _, op := range v.Keys() {
			arg, _ := v.Get(op)
			c, err := q.operator(column, op, arg)
			if err != nil {
				return "", err
			}
			conds = append(conds, c)
		}
		return strings.Join(conds, " and "), nil
	default:
		return column + " = " + q.arg(v), nil
	}
}

func (q *queryBuilder) operator(column, op string, arg any) (string, error) {
	simple := map[string]string{"$gt": ">", "$gte": ">=", "$lt": "<", "$lte": "<=", "$like": "like", "$ilike": "ilike", "$re": "~", "$contains": "@>", "$contained": "<@", "$overlap": "&&"}
	switch op {
	case "$eq":
		if arg == nil {
			return column + " is null", nil
		}
		if list, ok := arg.([]any); ok {
			return q.inList(column, dedupe(list), false), nil
		}
		return column + " = " + q.arg(arg), nil
	case "$ne":
		if arg == nil {
			return column + " is not null", nil
		}
		return column + " != " + q.arg(arg), nil
	case "$in", "$nin":
		list, ok := arg.([]any)
		if !ok {
			list = []any{arg}
		}
		return q.inList(column, dedupe(list), op == "$nin"), nil
	case "$exists":
		if b, _ := arg.(bool); b {
			return column + " is not null", nil
		}
		return column + " is null", nil
	case "$not":
		c, err := q.operators(column, arg)
		if err != nil {
			return "", err
		}
		return "not (" + c + ")", nil
	}
	if sqlOp, ok := simple[op]; ok {
		return column + " " + sqlOp + " " + q.arg(arg), nil
	}
	return "", fmt.Errorf("unsupported operator %s", op)
}

func dedupe(values []any) []any {
	seen := map[string]bool{}
	out := make([]any, 0, len(values))
	for _, v := range values {
		k := fmt.Sprintf("%T:%v", v, v)
		if !seen[k] {
			seen[k] = true
			out = append(out, v)
		}
	}
	return out
}

// joinRelation LEFT JOINs a relation used in a filter. Like MikroORM, the
// soft delete condition is not applied to tables joined for filtering.
func (q *queryBuilder) joinRelation(e *dml.Entity, alias, relation, path string) (string, *dml.Entity, error) {
	key := path + "." + relation
	j, err := q.schema.ResolveJoin(e, relation)
	if err != nil {
		return "", nil, err
	}
	if existing, ok := q.joinAliases[key]; ok {
		return existing, j.Target, nil
	}
	targetAlias := q.nextAlias()
	q.joinAliases[key] = targetAlias
	if j.Many {
		q.toManyJoins = true
	}
	if j.PivotTable != "" {
		pivotAlias := q.nextAlias()
		q.joins = append(q.joins,
			fmt.Sprintf(`left join %s as %s on %s = %s`, quote(j.PivotTable), quote(pivotAlias), col(alias, j.SourceColumns[0]), col(pivotAlias, j.PivotSourceColumns[0])),
			fmt.Sprintf(`left join %s as %s on %s = %s`, quote(j.Target.Meta.Collection), quote(targetAlias), col(pivotAlias, j.PivotTargetColumns[0]), col(targetAlias, j.TargetColumns[0])),
		)
	} else {
		q.joins = append(q.joins, fmt.Sprintf(`left join %s as %s on %s = %s`,
			quote(j.Target.Meta.Collection), quote(targetAlias), col(alias, j.SourceColumns[0]), col(targetAlias, j.TargetColumns[0])))
	}
	return targetAlias, j.Target, nil
}

// freeText mirrors the freeTextSearch filter: an ILIKE on every searchable
// property, recursing into searchable relations (LEFT JOINed with the soft
// delete condition).
func (q *queryBuilder) freeText(e *dml.Entity, alias, value, path string) (string, error) {
	conds, err := q.searchConditions(e, alias, "%"+value+"%", path, map[string]bool{e.Meta.ClassName: true}, false)
	if err != nil || len(conds) == 0 {
		return "", err
	}
	return "(" + strings.Join(conds, " or ") + ")", nil
}

func (q *queryBuilder) searchConditions(e *dml.Entity, alias, pattern, path string, visited map[string]bool, stop bool) ([]string, error) {
	var conds []string
	for _, p := range e.Properties {
		if p.Field == nil || p.Field.Computed || p.Field.DataType.Options == nil || !p.Field.DataType.Options.Searchable {
			continue
		}
		mp, ok := e.MetaProperty(p.Name)
		if !ok || mp.Kind != "scalar" {
			continue
		}
		column := col(alias, mp.FieldNames[0])
		if mp.ColumnType != "text" {
			column = "cast(" + column + " as text)"
		}
		conds = append(conds, column+" ilike "+q.arg(pattern))
	}
	if stop {
		return conds, nil
	}
	for _, p := range e.Properties {
		if p.Relation == nil || !p.Relation.Searchable {
			continue
		}
		target, err := q.schema.Entity(p.Relation.Entity)
		if err != nil {
			return nil, err
		}
		self := target.Meta.ClassName == e.Meta.ClassName
		if !self && visited[target.Meta.ClassName] {
			continue
		}
		branch := map[string]bool{target.Meta.ClassName: true}
		for k := range visited {
			branch[k] = true
		}
		targetAlias, err := q.joinSearchRelation(e, alias, p.Name, path)
		if err != nil {
			return nil, err
		}
		sub, err := q.searchConditions(target, targetAlias, pattern, path+"."+p.Name, branch, self)
		if err != nil {
			return nil, err
		}
		if len(sub) > 0 {
			conds = append(conds, "(("+strings.Join(sub, " or ")+"))")
		}
	}
	return conds, nil
}

func (q *queryBuilder) joinSearchRelation(e *dml.Entity, alias, relation, path string) (string, error) {
	key := "q:" + path + "." + relation
	if existing, ok := q.joinAliases[key]; ok {
		return existing, nil
	}
	j, err := q.schema.ResolveJoin(e, relation)
	if err != nil {
		return "", err
	}
	if j.Many {
		q.toManyJoins = true
	}
	targetAlias := q.nextAlias()
	q.joinAliases[key] = targetAlias
	soft := ""
	if !q.withDeleted && hasDeletedAt(j.Target) {
		soft = " and " + col(targetAlias, "deleted_at") + " is null"
	}
	if j.PivotTable != "" {
		pivotAlias := q.nextAlias()
		q.joins = append(q.joins,
			fmt.Sprintf(`left join %s as %s on %s = %s`, quote(j.PivotTable), quote(pivotAlias), col(alias, j.SourceColumns[0]), col(pivotAlias, j.PivotSourceColumns[0])),
			fmt.Sprintf(`left join %s as %s on %s = %s%s`, quote(j.Target.Meta.Collection), quote(targetAlias), col(pivotAlias, j.PivotTargetColumns[0]), col(targetAlias, j.TargetColumns[0]), soft),
		)
	} else {
		q.joins = append(q.joins, fmt.Sprintf(`left join %s as %s on %s = %s%s`,
			quote(j.Target.Meta.Collection), quote(targetAlias), col(alias, j.SourceColumns[0]), col(targetAlias, j.TargetColumns[0]), soft))
	}
	return targetAlias, nil
}
