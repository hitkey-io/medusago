package dal

import (
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// CrossModuleJoinLink mirrors CrossModuleJoinLink.
type CrossModuleJoinLink struct {
	Schema    string
	Table     string
	SourceKey string
	TargetKey string
}

// CrossModuleJoinTarget mirrors CrossModuleJoinTarget.
type CrossModuleJoinTarget struct {
	Schema     string
	Table      string
	PrimaryKey string
	Filters    *Record
}

// CrossModuleJoinSpec mirrors CrossModuleJoinSpec: a filter on a table of
// another module (same database) pushed into the root query as EXISTS.
type CrossModuleJoinSpec struct {
	Parent string
	Link   CrossModuleJoinLink
	Target CrossModuleJoinTarget
}

// RawFilter is a where condition written in SQL (raw() keys of a MikroORM
// filter): it receives the root alias and the placeholder allocator.
type RawFilter func(alias string, arg func(any) string) string

type resolvedJoin struct {
	*CrossModuleJoinSpec
	alias string
}

type correlateSpec struct {
	table, alias, key string
}

type joinSQLContext struct {
	linkAliasCounter int
	childrenByParent map[string][]*resolvedJoin
	withDeleted      bool
	arg              func(any) string
}

func quoteIdent(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

func qualifyTable(schema, table string) string {
	if schema == "" {
		schema = "public"
	}
	return quoteIdent(schema) + "." + quoteIdent(table)
}

func targetPK(j *resolvedJoin) string {
	if j.Target.PrimaryKey != "" {
		return j.Target.PrimaryKey
	}
	return "id"
}

// resolveCrossModuleJoins mirrors resolveCrossModuleJoins (aliases by
// target table, with validation).
func resolveCrossModuleJoins(joins []*CrossModuleJoinSpec) ([]*resolvedJoin, error) {
	tables := map[string]bool{}
	for _, j := range joins {
		if tables[j.Target.Table] {
			return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`Duplicate cross-module join target table "%s". Each join must target a unique table.`, j.Target.Table))
		}
		tables[j.Target.Table] = true
	}
	for _, j := range joins {
		if j.Parent == "" {
			continue
		}
		if j.Parent == j.Target.Table {
			return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`Cross-module join for "%s" cannot be its own parent.`, j.Target.Table))
		}
		if !tables[j.Parent] {
			return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`Cross-module join for "%s" references unknown parent target table "%s".`, j.Target.Table, j.Parent))
		}
	}
	used := map[string]int{}
	out := make([]*resolvedJoin, len(joins))
	for i, j := range joins {
		n := used[j.Target.Table]
		used[j.Target.Table] = n + 1
		alias := j.Target.Table
		if n > 0 {
			alias = fmt.Sprintf("%s_%d", j.Target.Table, n+1)
		}
		out[i] = &resolvedJoin{CrossModuleJoinSpec: j, alias: alias}
	}
	return out, nil
}

func inferLinkType(j *resolvedJoin, c correlateSpec) string {
	if j.Link.Schema == j.Target.Schema && j.Link.Table == j.Target.Table && j.Link.TargetKey == targetPK(j) {
		return "target"
	}
	if j.Link.Table == c.table && j.Link.SourceKey == c.key {
		return "source"
	}
	return "pivot"
}

func hasTargetFilters(j *resolvedJoin) bool { return j.Target.Filters != nil && j.Target.Filters.Len() > 0 }

func joinRequiresFilter(j *resolvedJoin, children map[string][]*resolvedJoin) bool {
	if hasTargetFilters(j) {
		return true
	}
	for _, c := range children[j.alias] {
		if joinRequiresFilter(c, children) {
			return true
		}
	}
	return false
}

// CrossModuleExistsFilters mirrors augmentFindOptionsWithCrossModuleJoins:
// one EXISTS condition per root join that filters. rootTable/primaryKey are
// the root entity table and key.
func CrossModuleExistsFilters(joins []*CrossModuleJoinSpec, rootTable, primaryKey string, withDeleted bool) ([]RawFilter, error) {
	if len(joins) == 0 {
		return nil, nil
	}
	resolved, err := resolveCrossModuleJoins(joins)
	if err != nil {
		return nil, err
	}
	aliasOf := map[string]string{}
	for _, j := range resolved {
		aliasOf[j.Target.Table] = j.alias
	}
	children := map[string][]*resolvedJoin{}
	for _, j := range resolved {
		if j.Parent == "" {
			continue
		}
		if pa, ok := aliasOf[j.Parent]; ok {
			children[pa] = append(children[pa], j)
		}
	}
	if primaryKey == "" {
		primaryKey = "id"
	}
	var out []RawFilter
	for _, j := range resolved {
		if j.Parent != "" || !joinRequiresFilter(j, children) {
			continue
		}
		join := j
		out = append(out, func(alias string, arg func(any) string) string {
			ctx := &joinSQLContext{childrenByParent: children, withDeleted: withDeleted, arg: arg}
			return buildExistsSQL(join, correlateSpec{table: rootTable, alias: alias, key: primaryKey}, ctx)
		})
	}
	return out, nil
}

func buildExistsSQL(j *resolvedJoin, c correlateSpec, ctx *joinSQLContext) string {
	linkType := inferLinkType(j, c)
	pk := targetPK(j)
	if linkType == "pivot" && j.Target.Filters != nil && j.Target.Filters.Len() == 1 && j.Target.Filters.Has(pk) {
		v, _ := j.Target.Filters.Get(pk)
		if v != nil {
			return buildTargetIDOnlyExists(j, c, ctx, v)
		}
	}
	if linkType == "pivot" {
		return buildPivotExists(j, c, ctx)
	}
	return buildNoLinkExists(j, c, ctx, linkType)
}

func (ctx *joinSQLContext) nextLinkAlias() string {
	a := fmt.Sprintf("cm_link_%d", ctx.linkAliasCounter)
	ctx.linkAliasCounter++
	return a
}

func softDeleteSQL(alias string, withDeleted bool) string {
	if withDeleted {
		return ""
	}
	return quoteIdent(alias) + `."deleted_at" is null`
}

func joinClauses(clauses []string) string {
	var kept []string
	for _, c := range clauses {
		if c != "" {
			kept = append(kept, c)
		}
	}
	return strings.Join(kept, " and ")
}

func buildTargetIDOnlyExists(j *resolvedJoin, c correlateSpec, ctx *joinSQLContext, idFilter any) string {
	link := ctx.nextLinkAlias()
	f := NewRecord()
	f.Set(j.Link.TargetKey, idFilter)
	clauses := []string{
		quoteIdent(link) + "." + quoteIdent(j.Link.SourceKey) + " = " + quoteIdent(c.alias) + "." + quoteIdent(c.key),
		softDeleteSQL(link, ctx.withDeleted),
		buildFilterSQL(link, f, ctx.arg),
	}
	return fmt.Sprintf("exists (select 1 from %s as %s where %s)", qualifyTable(j.Link.Schema, j.Link.Table), quoteIdent(link), joinClauses(clauses))
}

func buildPivotExists(j *resolvedJoin, c correlateSpec, ctx *joinSQLContext) string {
	link := ctx.nextLinkAlias()
	target := j.alias
	clauses := []string{
		quoteIdent(link) + "." + quoteIdent(j.Link.SourceKey) + " = " + quoteIdent(c.alias) + "." + quoteIdent(c.key),
		quoteIdent(link) + "." + quoteIdent(j.Link.TargetKey) + " = " + quoteIdent(target) + "." + quoteIdent(targetPK(j)),
		softDeleteSQL(link, ctx.withDeleted),
		softDeleteSQL(target, ctx.withDeleted),
	}
	clauses = append(clauses, targetFilterClauses(j, target, ctx)...)
	return fmt.Sprintf("exists (select 1 from %s as %s inner join %s as %s on true where %s)",
		qualifyTable(j.Link.Schema, j.Link.Table), quoteIdent(link), qualifyTable(j.Target.Schema, j.Target.Table), quoteIdent(target), joinClauses(clauses))
}

func buildNoLinkExists(j *resolvedJoin, c correlateSpec, ctx *joinSQLContext, linkType string) string {
	target := j.alias
	var corr string
	if linkType == "target" {
		corr = quoteIdent(target) + "." + quoteIdent(j.Link.SourceKey) + " = " + quoteIdent(c.alias) + "." + quoteIdent(c.key)
	} else {
		corr = quoteIdent(target) + "." + quoteIdent(targetPK(j)) + " = " + quoteIdent(c.alias) + "." + quoteIdent(j.Link.TargetKey)
	}
	clauses := []string{corr, softDeleteSQL(target, ctx.withDeleted)}
	clauses = append(clauses, targetFilterClauses(j, target, ctx)...)
	return fmt.Sprintf("exists (select 1 from %s as %s where %s)", qualifyTable(j.Target.Schema, j.Target.Table), quoteIdent(target), joinClauses(clauses))
}

func targetFilterClauses(j *resolvedJoin, target string, ctx *joinSQLContext) []string {
	var clauses []string
	if hasTargetFilters(j) {
		clauses = append(clauses, buildFilterSQL(target, j.Target.Filters, ctx.arg))
	}
	for _, child := range ctx.childrenByParent[j.alias] {
		if !joinRequiresFilter(child, ctx.childrenByParent) {
			continue
		}
		if s := buildExistsSQL(child, correlateSpec{table: j.Target.Table, alias: target, key: targetPK(j)}, ctx); s != "" {
			clauses = append(clauses, s)
		}
	}
	return clauses
}

// buildFilterSQL mirrors buildFilterSql.
func buildFilterSQL(alias string, filters *Record, arg func(any) string) string {
	var clauses []string
	for _, key := range filters.Keys() {
		value, _ := filters.Get(key)
		switch {
		case key == "$and":
			if list, ok := value.([]any); ok {
				for _, cond := range list {
					if r, ok := cond.(*Record); ok {
						if s := buildFilterSQL(alias, r, arg); s != "" {
							clauses = append(clauses, "("+s+")")
						}
					}
				}
				continue
			}
		case key == "$or":
			if list, ok := value.([]any); ok {
				var parts []string
				for _, cond := range list {
					if r, ok := cond.(*Record); ok {
						if s := buildFilterSQL(alias, r, arg); s != "" {
							parts = append(parts, "("+s+")")
						}
					}
				}
				if len(parts) > 0 {
					clauses = append(clauses, "("+strings.Join(parts, " or ")+")")
				}
				continue
			}
		}
		column := quoteIdent(alias) + "." + quoteIdent(key)
		if value == nil {
			clauses = append(clauses, column+" is null")
			continue
		}
		if list, ok := value.([]any); ok {
			if s := inArraySQL(column, list, false, arg); s != "" {
				clauses = append(clauses, s)
			}
			continue
		}
		if r, ok := value.(*Record); ok {
			hasOps := false
			for _, k := range r.Keys() {
				if strings.HasPrefix(k, "$") {
					hasOps = true
				}
			}
			if hasOps {
				for _, op := range r.Keys() {
					v, _ := r.Get(op)
					if s := operatorSQL(column, op, v, arg); s != "" {
						clauses = append(clauses, s)
					}
				}
				continue
			}
		}
		clauses = append(clauses, column+" = "+arg(value))
	}
	return strings.Join(clauses, " and ")
}

func inArraySQL(column string, values []any, negated bool, arg func(any) string) string {
	if len(values) == 0 {
		if negated {
			return ""
		}
		return "1 = 0"
	}
	ph := make([]string, len(values))
	for i, v := range values {
		ph[i] = arg(v)
	}
	array := "array[" + strings.Join(ph, ", ") + "]"
	if negated {
		return "not (" + column + " = any(" + array + "))"
	}
	return column + " = any(" + array + ")"
}

func operatorSQL(column, op string, value any, arg func(any) string) string {
	switch op {
	case "$eq":
		if value == nil {
			return column + " is null"
		}
		return column + " = " + arg(value)
	case "$ne":
		if value == nil {
			return column + " is not null"
		}
		return column + " <> " + arg(value)
	case "$in":
		list, _ := value.([]any)
		return inArraySQL(column, list, false, arg)
	case "$nin":
		list, _ := value.([]any)
		return inArraySQL(column, list, true, arg)
	case "$gt":
		return column + " > " + arg(value)
	case "$gte":
		return column + " >= " + arg(value)
	case "$lt":
		return column + " < " + arg(value)
	case "$lte":
		return column + " <= " + arg(value)
	case "$like":
		return column + " like " + arg(value)
	case "$ilike":
		return column + " ilike " + arg(value)
	case "$is":
		if value == nil {
			return column + " is null"
		}
		return column + " is " + arg(value)
	}
	return ""
}

// crossModuleOrder mirrors transformOrderByForCrossModuleJoins: an order on
// "<join alias>.<field>" becomes a correlated scalar subquery.
func crossModuleOrder(order []OrderBy, joins []*CrossModuleJoinSpec, rootTable, primaryKey string, withDeleted bool) []OrderBy {
	if len(order) == 0 {
		return order
	}
	resolved, err := resolveCrossModuleJoins(joins)
	if err != nil {
		return order
	}
	out := make([]OrderBy, len(order))
	for i, o := range order {
		out[i] = o
		parts := strings.SplitN(o.Field, ".", 2)
		if len(parts) < 2 || o.Raw != nil {
			continue
		}
		index := -1
		for k, j := range resolved {
			if j.alias == parts[0] {
				index = k
				break
			}
		}
		if index < 0 {
			continue
		}
		join, field := resolved[index], parts[1]
		out[i].Raw = func(alias string) string {
			return orderSubquery(join, correlateSpec{table: rootTable, alias: alias, key: primaryKey}, index, field, withDeleted)
		}
	}
	return out
}

func orderSubquery(j *resolvedJoin, c correlateSpec, index int, field string, withDeleted bool) string {
	linkType := inferLinkType(j, c)
	pk := targetPK(j)
	switch {
	case field == pk && linkType == "pivot":
		link := fmt.Sprintf("cm_order_link_%d", index)
		clauses := []string{
			quoteIdent(link) + "." + quoteIdent(j.Link.SourceKey) + " = " + quoteIdent(c.alias) + "." + quoteIdent(c.key),
			softDeleteSQL(link, withDeleted),
		}
		return fmt.Sprintf("(select %s.%s from %s as %s where %s order by %s.%s limit 1)", quoteIdent(link), quoteIdent(j.Link.TargetKey),
			qualifyTable(j.Link.Schema, j.Link.Table), quoteIdent(link), joinClauses(clauses), quoteIdent(link), quoteIdent(j.Link.TargetKey))
	case linkType == "pivot":
		link := fmt.Sprintf("cm_order_link_%d", index)
		target := j.alias
		clauses := []string{
			quoteIdent(link) + "." + quoteIdent(j.Link.SourceKey) + " = " + quoteIdent(c.alias) + "." + quoteIdent(c.key),
			quoteIdent(link) + "." + quoteIdent(j.Link.TargetKey) + " = " + quoteIdent(target) + "." + quoteIdent(pk),
			softDeleteSQL(link, withDeleted),
			softDeleteSQL(target, withDeleted),
		}
		return fmt.Sprintf("(select %s.%s from %s as %s inner join %s as %s on true where %s order by %s.%s limit 1)", quoteIdent(target), quoteIdent(field),
			qualifyTable(j.Link.Schema, j.Link.Table), quoteIdent(link), qualifyTable(j.Target.Schema, j.Target.Table), quoteIdent(target),
			joinClauses(clauses), quoteIdent(target), quoteIdent(pk))
	}
	target := j.alias
	var corr string
	if linkType == "target" {
		corr = quoteIdent(target) + "." + quoteIdent(j.Link.SourceKey) + " = " + quoteIdent(c.alias) + "." + quoteIdent(c.key)
	} else {
		corr = quoteIdent(target) + "." + quoteIdent(pk) + " = " + quoteIdent(c.alias) + "." + quoteIdent(j.Link.TargetKey)
	}
	clauses := []string{corr, softDeleteSQL(target, withDeleted)}
	return fmt.Sprintf("(select %s.%s from %s as %s where %s order by %s.%s limit 1)", quoteIdent(target), quoteIdent(field),
		qualifyTable(j.Target.Schema, j.Target.Table), quoteIdent(target), joinClauses(clauses), quoteIdent(target), quoteIdent(pk))
}
