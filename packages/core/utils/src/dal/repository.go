package dal

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

// OrderBy is one ordering criterion.
type OrderBy struct {
	Field     string
	Direction string // ASC or DESC
	// Raw orders by a SQL expression of the root alias (cross-module
	// ordering subqueries).
	Raw func(alias string) string
}

// FindOptions mirrors DAL.FindOptions as produced by buildQuery.
type FindOptions struct {
	Where       *Record
	Select      []string
	Relations   []string
	Order       []OrderBy
	Take        *int
	Skip        *int
	WithDeleted bool
	// PopulateWhere filters the rows loaded for a relation path (equality
	// and null checks).
	PopulateWhere map[string]*Record
	// JoinedStrategy mirrors MikroORM's JOINED load strategy: the foreign
	// keys joining partially selected collections are not hydrated.
	JoinedStrategy bool
	// CrossModuleJoins mirrors options.__internal.crossModuleJoins: filters
	// on tables of other modules pushed down as EXISTS conditions.
	CrossModuleJoins []*CrossModuleJoinSpec
}

// Repository reads the rows of one model, reproducing the queries and the
// serialization of the upstream MikroORM repository.
type Repository struct {
	DB     Querier
	Schema *Schema
	Entity *dml.Entity
}

type instance struct {
	entity    *dml.Entity
	data      Row
	relations map[string]any // *instance, []*instance or nil
	created   bool
	// sel is the field selection the instance was loaded with (nil: every
	// field); a reference found in the identity map serializes these.
	sel *fieldTree
	// initChanged lists the columns @OnInit hooks changed after hydration.
	initChanged map[string]bool
}

func (inst *instance) selectFields(t *fieldTree) {
	if t == nil {
		t = allFields
	}
	if inst.sel == nil || t.all {
		inst.sel = t
	}
}

type identityMap map[string]*instance

func pkValue(e *dml.Entity, row Row) string {
	parts := make([]string, len(e.Meta.PrimaryKey))
	for i, pk := range e.Meta.PrimaryKey {
		parts[i] = fmt.Sprint(row[pk])
	}
	return strings.Join(parts, "\x00")
}

func (m identityMap) get(e *dml.Entity, row Row) *instance {
	key := e.Meta.ClassName + "\x00" + pkValue(e, row)
	if inst, ok := m[key]; ok {
		// Like MikroORM's merge, columns loaded now that the managed
		// entity did not have yet are added to it.
		merged := false
		for k, v := range row {
			if _, has := inst.data[k]; !has {
				inst.data[k] = v
				merged = true
			}
		}
		if merged {
			m.initEntity(inst)
		}
		return inst
	}
	inst := &instance{entity: e, data: row, relations: map[string]any{}}
	m.initEntity(inst)
	m[key] = inst
	return inst
}

type sharedIdentityMapKey struct{}

type noRelationRefreshKey struct{}

// WithoutRelationRefresh makes finds keep the relations already initialized
// on the shared identity map entities (MikroORM refresh: false).
func WithoutRelationRefresh(ctx context.Context) context.Context {
	return context.WithValue(WithSharedIdentityMap(ctx), noRelationRefreshKey{}, true)
}

func relationRefreshDisabled(ctx context.Context) bool {
	v, _ := ctx.Value(noRelationRefreshKey{}).(bool)
	return v
}

// WithSharedIdentityMap returns a context whose finds share one identity
// map, like the queries of one MikroORM entity manager (a service method
// decorated with @InjectManager).
func WithSharedIdentityMap(ctx context.Context) context.Context {
	if _, ok := ctx.Value(sharedIdentityMapKey{}).(identityMap); ok {
		return ctx
	}
	return context.WithValue(ctx, sharedIdentityMapKey{}, identityMap{})
}

func sharedIdentityMap(ctx context.Context) identityMap {
	if m, ok := ctx.Value(sharedIdentityMapKey{}).(identityMap); ok {
		return m
	}
	return identityMap{}
}

func (m identityMap) lookup(e *dml.Entity, pk any) (*instance, bool) {
	inst, ok := m[e.Meta.ClassName+"\x00"+fmt.Sprint(pk)]
	return inst, ok
}

func quote(id string) string { return `"` + strings.ReplaceAll(id, `"`, `""`) + `"` }

func col(alias, column string) string { return quote(alias) + "." + quote(column) }

func hasDeletedAt(e *dml.Entity) bool {
	_, ok := e.MetaProperty("deleted_at")
	return ok
}

// Find mirrors repository.find.
func (r *Repository) Find(ctx context.Context, opts FindOptions) ([]*Record, error) {
	opts, err := r.augmentCrossModuleJoins(opts)
	if err != nil {
		return nil, err
	}
	opts = r.pruneFindOptions(opts)
	roots, idm, err := r.find(ctx, opts)
	if err != nil {
		return nil, err
	}
	if err := r.loadInverseOneToOneRefs(ctx, idm); err != nil {
		return nil, err
	}
	return r.serialize(roots, idm, opts)
}

// FindAndCount mirrors repository.findAndCount.
func (r *Repository) FindAndCount(ctx context.Context, opts FindOptions) ([]*Record, int, error) {
	opts, err := r.augmentCrossModuleJoins(opts)
	if err != nil {
		return nil, 0, err
	}
	records, err := r.Find(ctx, opts)
	if err != nil {
		return nil, 0, err
	}
	count, err := r.count(ctx, opts)
	return records, count, err
}

// augmentCrossModuleJoins mirrors augmentFindOptionsWithCrossModuleJoins:
// the root where becomes { $and: [where, ...exists] }.
func (r *Repository) augmentCrossModuleJoins(opts FindOptions) (FindOptions, error) {
	if len(opts.CrossModuleJoins) == 0 {
		return opts, nil
	}
	pk := "id"
	if len(r.Entity.Meta.PrimaryKey) > 0 {
		pk = r.Entity.Meta.PrimaryKey[0]
	}
	filters, err := CrossModuleExistsFilters(opts.CrossModuleJoins, r.Entity.Meta.Collection, pk, opts.WithDeleted)
	if err != nil {
		return opts, err
	}
	opts.Order = crossModuleOrder(opts.Order, opts.CrossModuleJoins, r.Entity.Meta.Collection, pk, opts.WithDeleted)
	opts.CrossModuleJoins = nil
	if len(filters) == 0 {
		return opts, nil
	}
	var and []any
	if opts.Where != nil && opts.Where.Len() > 0 {
		and = append(and, opts.Where)
	}
	for i, f := range filters {
		cond := NewRecord()
		cond.Set(fmt.Sprintf("__cross_module_%d", i), f)
		and = append(and, cond)
	}
	where := NewRecord()
	where.Set("$and", and)
	opts.Where = where
	return opts, nil
}

func (r *Repository) count(ctx context.Context, opts FindOptions) (int, error) {
	qb := newQueryBuilder(r.Schema, opts.WithDeleted)
	where, err := qb.rootWhere(r.Entity, "e0", opts.Where)
	if err != nil {
		return 0, err
	}
	selectExpr := "count(*)"
	if qb.toManyJoins {
		pk := make([]string, len(r.Entity.Meta.PrimaryKey))
		for i, k := range r.Entity.Meta.PrimaryKey {
			pk[i] = col("e0", k)
		}
		selectExpr = "count(distinct(" + strings.Join(pk, ", ") + "))"
	}
	sql := fmt.Sprintf(`select %s as "count" from %s as "e0"%s%s`, selectExpr, quote(r.Entity.Meta.Collection), qb.joinSQL(), where)
	rows, err := queryRows(ctx, r.DB, sql, qb.args...)
	if err != nil {
		return 0, err
	}
	var n int
	fmt.Sscan(fmt.Sprint(rows[0]["count"]), &n)
	return n, nil
}

func (r *Repository) find(ctx context.Context, opts FindOptions) ([]*instance, identityMap, error) {
	r.autoFlush(r.Entity)
	opts = r.pruneFindOptions(opts)
	// Orders on relation fields ("images.rank") order the select-in query
	// loading that relation; the root query keeps the orders of its own
	// fields.
	relationOrder := map[string][]OrderBy{}
	var rootOrder []OrderBy
	for _, o := range opts.Order {
		if i := strings.LastIndex(o.Field, "."); i > 0 && o.Raw == nil {
			relationOrder[o.Field[:i]] = append(relationOrder[o.Field[:i]], OrderBy{Field: o.Field[i+1:], Direction: o.Direction})
			continue
		}
		rootOrder = append(rootOrder, o)
	}
	opts.Order = rootOrder
	qb := newQueryBuilder(r.Schema, opts.WithDeleted)
	where, err := qb.rootWhere(r.Entity, "e0", opts.Where)
	if err != nil {
		return nil, nil, err
	}
	order, err := r.orderSQL(opts.Order, "e0")
	if err != nil {
		return nil, nil, err
	}

	// Like the BALANCED load strategy, populated to-one relations are LEFT
	// JOINed in the root query (with the soft delete condition in the join).
	// Their data is hydrated from the select-in queries below; the joins keep
	// the root query (and so the order of rows with equal sort keys) the same
	// as upstream.
	tree := buildFieldTree(opts, r.relationPaths(opts))
	toOneJoins, joinedColumns := r.toOneJoins(r.relationPaths(opts), tree, opts.WithDeleted)
	if toOneJoins != "" {
		qb.joins = append([]string{toOneJoins}, qb.joins...)
	}
	rootColumns := `"e0".*`
	if !tree.all {
		cols := selectedColumns(r.Entity, tree)
		quoted := make([]string, len(cols))
		for i, c := range cols {
			quoted[i] = col("e0", c)
		}
		quoted = append(quoted, formulaColumns(r.Entity.Meta.ClassName, "e0", tree)...)
		rootColumns = strings.Join(quoted, ", ")
	}
	if len(joinedColumns) > 0 {
		rootColumns += ", " + strings.Join(joinedColumns, ", ")
	}

	table := quote(r.Entity.Meta.Collection)
	paging := ""
	if opts.Take != nil {
		paging += fmt.Sprintf(" limit %d", *opts.Take)
	}
	if opts.Skip != nil && *opts.Skip > 0 {
		paging += fmt.Sprintf(" offset %d", *opts.Skip)
	}

	var sql string
	if qb.toManyJoins && paging != "" {
		// Like MikroORM: paginate the distinct roots in a sub-query.
		pk := col("e0", r.Entity.Meta.PrimaryKey[0])
		innerOrder := ""
		if len(opts.Order) > 0 {
			parts := make([]string, len(opts.Order))
			for i, o := range opts.Order {
				c, err := r.orderColumn(o.Field, "e0")
				if err != nil {
					return nil, nil, err
				}
				fn := "min"
				if strings.EqualFold(o.Direction, "DESC") {
					fn = "max"
				}
				parts[i] = fmt.Sprintf("%s(%s) %s", fn, c, strings.ToLower(o.Direction))
			}
			innerOrder = " order by " + strings.Join(parts, ", ")
		}
		sql = fmt.Sprintf(`select "e0".* from %s as "e0" where %s in (select %s from (select %s from %s as "e0"%s%s group by %s%s%s) as "e0")%s`,
			table, pk, pk, pk, table, qb.joinSQL(), where, pk, innerOrder, paging, order)
	} else {
		sql = fmt.Sprintf(`select %s from %s as "e0"%s%s%s%s`, rootColumns, table, qb.joinSQL(), where, order, paging)
	}

	rows, err := queryRows(ctx, r.DB, sql, qb.args...)
	if err != nil {
		return nil, nil, err
	}

	idm := sharedIdentityMap(ctx)
	var roots []*instance
	seen := map[*instance]bool{}
	for _, row := range rows {
		inst := idm.get(r.Entity, row)
		inst.selectFields(tree)
		if !seen[inst] {
			seen[inst] = true
			roots = append(roots, inst)
		}
	}

	if err := r.populate(ctx, roots, idm, r.relationPaths(opts), opts.WithDeleted, relationOrder, opts.PopulateWhere, tree); err != nil {
		return nil, nil, err
	}
	return roots, idm, nil
}

// toOneJoins builds the LEFT JOINs of the populated to-one relation chains
// starting at the root.
func (r *Repository) toOneJoins(paths []string, tree *fieldTree, withDeleted bool) (string, []string) {
	var joins, columns []string
	aliases := map[string]string{"": "e0"}
	entities := map[string]*dml.Entity{"": r.Entity}
	trees := map[string]*fieldTree{"": tree}
	n := 0
	for _, path := range paths {
		parent, rel := "", path
		if i := strings.LastIndex(path, "."); i >= 0 {
			parent, rel = path[:i], path[i+1:]
		}
		parentAlias, ok := aliases[parent]
		if !ok {
			continue
		}
		j, err := r.Schema.ResolveJoin(entities[parent], rel)
		if err != nil || j.Many || j.PivotTable != "" {
			continue
		}
		if j.Relation.Kind == "1:1" && !j.Relation.Owner {
			continue
		}
		n++
		alias := fmt.Sprintf("t%d", n)
		cond := fmt.Sprintf("%s = %s", col(parentAlias, j.SourceColumns[0]), col(alias, j.TargetColumns[0]))
		if !withDeleted && hasDeletedAt(j.Target) {
			cond += " and " + col(alias, "deleted_at") + " is null"
		}
		joins = append(joins, fmt.Sprintf("left join %s as %s on %s", quote(j.Target.Meta.Collection), quote(alias), cond))
		aliases[path] = alias
		entities[path] = j.Target
		sub := trees[parent].children[rel]
		if sub == nil {
			sub = allFields
		}
		trees[path] = sub
		// Like MikroORM, the joined entity columns are selected (aliased), which
		// keeps the join in the plan.
		for _, c := range selectedColumns(j.Target, sub) {
			columns = append(columns, fmt.Sprintf("%s as %s", col(alias, c), quote(alias+"__"+c)))
		}
	}
	return strings.Join(joins, " "), columns
}

// selectedColumns lists the columns MikroORM selects for an entity: every
// persisted column without a field selection, otherwise the primary keys,
// the selected properties (with the raw_ column of BigNumber fields) and the
// foreign keys of populated relations.
func selectedColumns(e *dml.Entity, tree *fieldTree) []string {
	var out []string
	seen := map[string]bool{}
	add := func(c string) {
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	bigNumbers := bigNumberFields(e)
	if tree == nil || tree.all {
		for _, p := range e.Meta.Properties {
			if (p.Persist != nil && !*p.Persist) || len(p.FieldNames) == 0 || p.Kind == "1:m" || p.Kind == "m:n" || (p.Kind == "1:1" && !p.Owner) {
				continue
			}
			add(p.FieldNames[0])
		}
		return out
	}
	for _, pk := range e.Meta.PrimaryKey {
		add(fieldColumn(e, pk))
	}
	for _, p := range e.Meta.Properties {
		if len(p.FieldNames) == 0 {
			continue
		}
		if p.Persist != nil && !*p.Persist {
			// a non persisted m:1 (its column belongs to the <name>_id
			// property) still selects its foreign key
			if tree.fields[p.Name] && (p.Kind == "m:1" || (p.Kind == "1:1" && p.Owner) || p.Kind == "scalar") {
				for _, fn := range p.FieldNames {
					add(fn)
				}
			}
			continue
		}
		if tree.fields[p.Name] {
			switch {
			case p.Kind == "1:m" || p.Kind == "m:n" || (p.Kind == "1:1" && !p.Owner):
				// a collection named in the selection: nothing to select,
				// it stays uninitialized (omitted from the result)
			case p.Kind == "m:1" || p.Kind == "1:1":
				for _, fn := range p.FieldNames {
					add(fn)
				}
			default:
				add(p.FieldNames[0])
				if bigNumbers[p.Name] {
					add("raw_" + p.Name)
				}
			}
		}
	}
	for rel := range tree.children {
		if p, ok := e.MetaProperty(rel); ok && (p.Kind == "m:1" || (p.Kind == "1:1" && p.Owner)) {
			for _, fn := range p.FieldNames {
				add(fn)
			}
		}
	}
	return out
}

func (r *Repository) orderColumn(field, alias string) (string, error) {
	prop, ok := r.Entity.MetaProperty(field)
	if !ok || len(prop.FieldNames) == 0 {
		return "", fmt.Errorf("cannot order %s by %s", r.Entity.Meta.ClassName, field)
	}
	return col(alias, prop.FieldNames[0]), nil
}

func (r *Repository) orderSQL(order []OrderBy, alias string) (string, error) {
	if len(order) == 0 {
		return "", nil
	}
	parts := make([]string, len(order))
	for i, o := range order {
		if o.Raw != nil {
			parts[i] = o.Raw(alias) + " " + strings.ToLower(o.Direction)
			continue
		}
		c, err := r.orderColumn(o.Field, alias)
		if err != nil {
			return "", err
		}
		parts[i] = c + " " + strings.ToLower(o.Direction)
	}
	return " order by " + strings.Join(parts, ", "), nil
}

// relationPaths returns the relations to populate: the explicit ones plus
// those implied by dotted selected fields, parents first.
func (r *Repository) relationPaths(opts FindOptions) []string {
	set := map[string]bool{}
	add := func(path string) {
		parts := strings.Split(path, ".")
		for i := range parts {
			set[strings.Join(parts[:i+1], ".")] = true
		}
	}
	for _, rel := range opts.Relations {
		add(rel)
	}
	for _, f := range opts.Select {
		if i := strings.LastIndex(f, "."); i > 0 {
			add(f[:i])
		}
	}
	paths := make([]string, 0, len(set))
	for p := range set {
		paths = append(paths, p)
	}
	sort.SliceStable(paths, func(i, j int) bool {
		di, dj := strings.Count(paths[i], "."), strings.Count(paths[j], ".")
		if di != dj {
			return di < dj
		}
		return paths[i] < paths[j]
	})
	return paths
}

// populate loads the relation paths level by level, the way MikroORM's
// select-in loading does.
func (r *Repository) populate(ctx context.Context, roots []*instance, idm identityMap, paths []string, withDeleted bool, relationOrder map[string][]OrderBy, populateWhere map[string]*Record, tree *fieldTree) error {
	byPath := map[string][]*instance{"": roots}
	for _, path := range paths {
		parentPath, relation := "", path
		if i := strings.LastIndex(path, "."); i >= 0 {
			parentPath, relation = path[:i], path[i+1:]
		}
		parents := byPath[parentPath]
		if len(parents) == 0 {
			byPath[path] = nil
			continue
		}
		children, err := r.loadRelationSelecting(ctx, parents, relation, idm, withDeleted, relationOrder[path], tree.at(path), populateWhere[path])
		if err != nil {
			return fmt.Errorf("populate %s: %w", path, err)
		}
		for _, c := range children {
			c.selectFields(tree.at(path))
		}
		byPath[path] = children
	}
	return nil
}

func uniqueValues(parents []*instance, columns []string) []any {
	var out []any
	seen := map[string]bool{}
	for _, p := range parents {
		v := p.data[columns[0]]
		if v == nil {
			continue
		}
		k := fmt.Sprint(v)
		if !seen[k] {
			seen[k] = true
			out = append(out, v)
		}
	}
	return out
}

func (r *Repository) loadRelation(ctx context.Context, parents []*instance, relation string, idm identityMap, withDeleted bool, order []OrderBy, where ...*Record) ([]*instance, error) {
	return r.loadRelationSelecting(ctx, parents, relation, idm, withDeleted, order, nil, where...)
}

// at returns the subtree of a relation path (nil when absent).
func (t *fieldTree) at(path string) *fieldTree {
	n := t
	for _, part := range strings.Split(path, ".") {
		if n == nil {
			return nil
		}
		n = n.children[part]
	}
	return n
}

// loadRelationSelecting loads a relation; the lazy formulas requested in
// sel are selected along with the columns.
func (r *Repository) loadRelationSelecting(ctx context.Context, parents []*instance, relation string, idm identityMap, withDeleted bool, order []OrderBy, sel *fieldTree, where ...*Record) ([]*instance, error) {
	// Like MikroORM's EntityLoader (refresh: false), a relation already
	// initialized on an entity of the identity map is not reloaded; its
	// entities still take part in the nested populate. Opt-in
	// (WithoutRelationRefresh): our writes do not add created entities to
	// the loaded collections like MikroORM does.
	if !relationRefreshDisabled(ctx) {
		return r.loadRelationRows(ctx, parents, relation, idm, withDeleted, order, sel, where...)
	}
	var pending []*instance
	var kept []*instance
	keptSeen := map[*instance]bool{}
	for _, p := range parents {
		v, initialized := p.relations[relation]
		if !initialized {
			pending = append(pending, p)
			continue
		}
		switch t := v.(type) {
		case []*instance:
			for _, c := range t {
				if !keptSeen[c] {
					keptSeen[c] = true
					kept = append(kept, c)
				}
			}
		case *instance:
			if t != nil && !keptSeen[t] {
				keptSeen[t] = true
				kept = append(kept, t)
			}
		}
	}
	if len(pending) == 0 {
		return kept, nil
	}
	loaded, err := r.loadRelationRows(ctx, pending, relation, idm, withDeleted, order, sel, where...)
	if err != nil || len(kept) == 0 {
		return loaded, err
	}
	for _, c := range loaded {
		if !keptSeen[c] {
			keptSeen[c] = true
			kept = append(kept, c)
		}
	}
	return kept, nil
}

func (r *Repository) loadRelationRows(ctx context.Context, parents []*instance, relation string, idm identityMap, withDeleted bool, order []OrderBy, sel *fieldTree, where ...*Record) ([]*instance, error) {
	owner := parents[0].entity
	join, err := r.Schema.ResolveJoin(owner, relation)
	if err != nil {
		return nil, err
	}
	target := join.Target
	softDelete := !withDeleted && hasDeletedAt(target)
	var args []any
	placeholders := func(values []any) string {
		ph := make([]string, len(values))
		for i, v := range values {
			args = append(args, v)
			ph[i] = fmt.Sprintf("$%d", len(args))
		}
		return strings.Join(ph, ", ")
	}

	var children []*instance
	seen := map[*instance]bool{}
	collect := func(inst *instance) {
		if !seen[inst] {
			seen[inst] = true
			children = append(children, inst)
		}
	}

	switch {
	case join.PivotTable != "":
		values := uniqueValues(parents, join.SourceColumns)
		for _, p := range parents {
			p.relations[relation] = []*instance{}
		}
		if len(values) == 0 {
			return nil, nil
		}
		cond := ""
		if softDelete {
			cond = ` and "t1"."deleted_at" is null`
		}
		order := ""
		if pivot, err := r.Schema.Entity(join.Relation.PivotEntity); err == nil && join.Relation.PivotEntity != "" {
			order = fmt.Sprintf(` order by "p0".%s asc`, quote(pivot.Meta.PrimaryKey[0]))
		}
		sql := fmt.Sprintf(`select "p0".%s as "__pivot_source", "t1".*%s from %s as "p0" inner join %s as "t1" on "p0".%s = "t1".%s where "p0".%s in (%s)%s%s`,
			quote(join.PivotSourceColumns[0]), extraColumns(formulaColumns(target.Meta.ClassName, "t1", sel)), quote(join.PivotTable), quote(target.Meta.Collection),
			quote(join.PivotTargetColumns[0]), quote(join.TargetColumns[0]),
			quote(join.PivotSourceColumns[0]), placeholders(values), cond, order)
		rows, err := queryRows(ctx, r.DB, sql, args...)
		if err != nil {
			return nil, err
		}
		bySource := map[string][]*instance{}
		for _, row := range rows {
			source := fmt.Sprint(row["__pivot_source"])
			delete(row, "__pivot_source")
			inst := idm.get(target, row)
			bySource[source] = append(bySource[source], inst)
			collect(inst)
		}
		for _, p := range parents {
			if list, ok := bySource[fmt.Sprint(p.data[join.SourceColumns[0]])]; ok {
				p.relations[relation] = list
			}
		}

	default:
		values := uniqueValues(parents, join.SourceColumns)
		if join.Many {
			for _, p := range parents {
				p.relations[relation] = []*instance{}
			}
		} else {
			for _, p := range parents {
				p.relations[relation] = nil
			}
		}
		if len(values) == 0 {
			return nil, nil
		}
		cond := fmt.Sprintf(`"t0".%s in (%s)`, quote(join.TargetColumns[0]), placeholders(values))
		if softDelete {
			cond = `"t0"."deleted_at" is null and ` + cond
		}
		if len(where) > 0 && where[0] != nil {
			for _, k := range where[0].Keys() {
				v, _ := where[0].Get(k)
				if raw, ok := v.(RawCondition); ok {
					cond += " and " + raw(`"t0"`)
					continue
				}
				column := k
				if p, ok := target.MetaProperty(k); ok && len(p.FieldNames) > 0 {
					column = p.FieldNames[0]
				}
				if v == nil {
					cond += fmt.Sprintf(` and "t0".%s is null`, quote(column))
				} else {
					cond += fmt.Sprintf(` and "t0".%s = %s`, quote(column), placeholders([]any{v}))
				}
			}
		}
		sql := fmt.Sprintf(`select "t0".*%s from %s as "t0" where %s`, extraColumns(formulaColumns(target.Meta.ClassName, "t0", sel)), quote(target.Meta.Collection), cond)
		if len(order) > 0 {
			tr := &Repository{DB: r.DB, Schema: r.Schema, Entity: target}
			o, err := tr.orderSQL(order, "t0")
			if err != nil {
				return nil, err
			}
			sql += o
		}
		rows, err := queryRows(ctx, r.DB, sql, args...)
		if err != nil {
			return nil, err
		}
		byKey := map[string][]*instance{}
		for _, row := range rows {
			inst := idm.get(target, row)
			k := fmt.Sprint(row[join.TargetColumns[0]])
			byKey[k] = append(byKey[k], inst)
			collect(inst)
		}
		for _, p := range parents {
			list := byKey[fmt.Sprint(p.data[join.SourceColumns[0]])]
			if join.Many {
				if list != nil {
					p.relations[relation] = list
				}
			} else if len(list) > 0 {
				p.relations[relation] = list[0]
			}
		}
	}
	return children, nil
}

// RawCondition is a populateWhere condition written in SQL; it receives the
// quoted alias of the loaded relation (MikroORM raw fragments with an alias
// callback).
type RawCondition func(alias string) string

func extraColumns(cols []string) string {
	if len(cols) == 0 {
		return ""
	}
	return ", " + strings.Join(cols, ", ")
}

// fieldTree is the selection for one entity level.
type fieldTree struct {
	all      bool
	joined   bool
	fields   map[string]bool
	children map[string]*fieldTree
}

func buildFieldTree(opts FindOptions, paths []string) *fieldTree {
	root := &fieldTree{all: len(opts.Select) == 0, fields: map[string]bool{}, children: map[string]*fieldTree{}}
	node := func(path string) *fieldTree {
		n := root
		if path == "" {
			return n
		}
		for _, part := range strings.Split(path, ".") {
			c, ok := n.children[part]
			if !ok {
				c = &fieldTree{fields: map[string]bool{}, children: map[string]*fieldTree{}}
				n.children[part] = c
			}
			n = c
		}
		return n
	}
	for _, p := range paths {
		node(p)
	}
	for _, f := range opts.Select {
		parent, name := "", f
		if i := strings.LastIndex(f, "."); i >= 0 {
			parent, name = f[:i], f[i+1:]
		}
		n := node(parent)
		if name == "*" {
			n.all = true
		} else {
			n.fields[name] = true
		}
	}
	// A populated relation without specific fields selects all its fields.
	var fill func(n *fieldTree)
	fill = func(n *fieldTree) {
		for _, c := range n.children {
			if len(c.fields) == 0 {
				c.all = true
			}
			fill(c)
		}
	}
	fill(root)
	if opts.JoinedStrategy {
		var mark func(n *fieldTree)
		mark = func(n *fieldTree) {
			n.joined = true
			for _, c := range n.children {
				mark(c)
			}
		}
		mark(root)
	}
	return root
}

func (r *Repository) serialize(roots []*instance, idm identityMap, opts FindOptions) ([]*Record, error) {
	tree := buildFieldTree(opts, r.relationPaths(opts))
	out := make([]*Record, 0, len(roots))
	for _, inst := range roots {
		rec, err := r.serializeInstance(inst, tree, nil, nil, idm)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

func bigNumberFields(e *dml.Entity) map[string]bool {
	out := map[string]bool{}
	for _, p := range e.Properties {
		if p.Field != nil && p.Field.DataType.Name == "bigNumber" {
			out[p.Name] = true
		}
	}
	return out
}

var allFields = &fieldTree{all: true, fields: map[string]bool{}, children: map[string]*fieldTree{}}

// serializeInstance mirrors EntitySerializer.serialize from
// mikro-orm-serializer.ts:
//   - keys: primary keys, then the hydrated properties (scalars, foreign keys
//     and references in metadata order, then initialized collections), then
//     the remaining metadata properties (BigNumber getters);
//   - a relation whose target class is one of the parents being serialized
//     is skipped (preventCircularRef);
//   - an unpopulated reference is serialized as {pk}, or fully when the
//     entity is initialized in the identity map; uninitialized collections
//     are omitted.
//
// joinFK lists the foreign key columns selected to join this entity to its
// parent; they are always part of the hydrated data.
func (r *Repository) serializeInstance(inst *instance, tree *fieldTree, parents []string, joinFK []string, idm identityMap) (*Record, error) {
	e := inst.entity
	rec := NewRecord()
	bigNumbers := bigNumberFields(e)
	childParents := append(append([]string(nil), parents...), e.Meta.ClassName)

	hydrated := func(name string) bool {
		return tree.all || tree.fields[name] || slices.Contains(e.Meta.PrimaryKey, name)
	}
	// fkHydrated reports whether the foreign key column of an owning
	// to-one relation was selected (by its property or a shadow property).
	fkHydrated := func(rel *dml.MetaProperty) bool {
		if tree.all {
			return true
		}
		if len(rel.FieldNames) == 0 || (rel.Kind == "1:1" && !rel.Owner) {
			return true
		}
		for _, mp := range e.Meta.Properties {
			if mp.Name != rel.Name && len(mp.FieldNames) > 0 && mp.FieldNames[0] == rel.FieldNames[0] && tree.fields[mp.Name] {
				return true
			}
		}
		return false
	}
	requiredFK := map[string]bool{}
	for _, fk := range joinFK {
		requiredFK[fk] = true
	}
	for relName := range tree.children {
		if prop, ok := e.MetaProperty(relName); ok && (prop.Kind == "m:1" || (prop.Kind == "1:1" && prop.Owner)) {
			for _, fn := range prop.FieldNames {
				requiredFK[fn] = true
			}
		}
	}
	isParent := func(p *dml.MetaProperty) bool {
		return !p.MapToPk && slices.Contains(parents, p.Entity)
	}

	toOne := func(p *dml.MetaProperty) error {
		if child, populated := tree.children[p.Name]; populated {
			v, err := r.serializeValue(inst.relations[p.Name], child, p, childParents, idm)
			if err != nil {
				return err
			}
			// the JOINED strategy assigns null to an unmatched to-one
			// relation (mapJoinedProps)
			if v == nil && p.Kind == "1:1" && !fkHydrated(p) && !tree.joined {
				// joined 1:1 without a match and without its foreign key
				// selected: MikroORM never assigns the property (m:1
				// relations always fetch their foreign key)
				return nil
			}
			rec.Set(p.Name, v)
			return nil
		}
		if inst.created || len(p.FieldNames) == 0 || !(hydrated(p.Name) || hydrated(p.FieldNames[0])) {
			return nil
		}
		fk := inst.data[p.FieldNames[0]]
		if p.Kind == "1:1" && !p.Owner {
			fk = inst.data[inverseRefKey(p.Name)]
		}
		if fk == nil {
			rec.Set(p.Name, nil)
			return nil
		}
		target, err := r.Schema.Entity(p.Entity)
		if err != nil {
			return err
		}
		if loaded, ok := idm.lookup(target, fk); ok {
			sel := loaded.sel
			if sel == nil {
				sel = allFields
			}
			ref, err := r.serializeInstance(loaded, sel, childParents, nil, idm)
			if err != nil {
				return err
			}
			rec.Set(p.Name, ref)
		} else {
			ref := NewRecord()
			ref.Set(target.Meta.PrimaryKey[0], fk)
			// formula properties of a reference are initialized to null
			for _, f := range formulas[target.Meta.ClassName] {
				ref.Set(f.Property, nil)
			}
			rec.Set(p.Name, ref)
		}
		return nil
	}
	for _, pk := range e.Meta.PrimaryKey {
		rec.Set(pk, inst.data[pk])
	}

	var collections []*dml.MetaProperty
	var lateScalars []*dml.MetaProperty
	var deferredRefs []*dml.MetaProperty
	for i := range e.Meta.Properties {
		p := &e.Meta.Properties[i]
		for _, f := range formulas[e.Meta.ClassName] {
			if f.Before == p.Name && tree.fields[f.Property] {
				if v, ok := inst.data[f.Property]; ok {
					rec.Set(f.Property, formulaValue(v))
				}
			}
		}
		switch {
		case p.Kind == "scalar":
			if bigNumbers[p.Name] || !hydrated(p.Name) {
				continue
			}
			if p.Persist != nil && !*p.Persist {
				// the shadow foreign key of a 1:1 owner is assigned after
				// hydration: it follows the hydrated properties
				lateScalars = append(lateScalars, p)
				continue
			}
			rec.Set(p.Name, runtimeNumber(p.Type, inst.data[p.FieldNames[0]]))

		case p.MapToPk && (p.Persist == nil || *p.Persist):
			// Foreign key property (e.g. product_id).
			if hydrated(p.Name) || requiredFK[p.FieldNames[0]] {
				rec.Set(p.Name, inst.data[p.FieldNames[0]])
			}

		case p.Kind == "m:1" || p.Kind == "1:1":
			if isParent(p) {
				continue
			}
			if tree.joined && p.Persist != nil && !*p.Persist && p.InversedBy == "" && p.MappedBy == "" {
				// JOINED strategy: unidirectional non persisted references
				// are assigned after the hydration of the scalar properties
				deferredRefs = append(deferredRefs, p)
				continue
			}
			if err := toOne(p); err != nil {
				return nil, err
			}

		default: // 1:m, m:n
			if _, populated := tree.children[p.Name]; populated && !isParent(p) {
				collections = append(collections, p)
			}
		}
	}

	for _, p := range lateScalars {
		v := inst.data[p.FieldNames[0]]
		if v == nil {
			// assigned from the relation, which is null
			continue
		}
		rec.Set(p.Name, runtimeNumber(p.Type, v))
	}

	for _, p := range deferredRefs {
		if err := toOne(p); err != nil {
			return nil, err
		}
	}

	for _, p := range collections {
		v, err := r.serializeValue(inst.relations[p.Name], tree.children[p.Name], p, childParents, idm)
		if err != nil {
			return nil, err
		}
		rec.Set(p.Name, v)
	}

	// BigNumber values: the computed ones (own properties set by entity
	// hooks) come before the getters of the persisted ones.
	bigNumberProps := make([]dml.Property, 0, len(e.Properties))
	for _, p := range e.Properties {
		if p.Field != nil && p.Field.Computed {
			bigNumberProps = append(bigNumberProps, p)
		}
	}
	for _, p := range e.Properties {
		if p.Field == nil || !p.Field.Computed {
			bigNumberProps = append(bigNumberProps, p)
		}
	}
	for _, p := range bigNumberProps {
		computed := p.Field != nil && p.Field.Computed
		// values set by entity hooks are serialized whatever the selection
		if !bigNumbers[p.Name] || (!computed && !hydrated(p.Name)) {
			continue
		}
		raw, ok := inst.data["raw_"+p.Name].(*Record)
		if !ok {
			if p.Field != nil && p.Field.Computed {
				// a computed value the entity hooks did not set: undefined
				continue
			}
			rec.Set(p.Name, nil)
			continue
		}
		value, _ := raw.Get("value")
		precision, _ := raw.Get("precision")
		bn, err := totals.NewBigNumberWithPrecision(totals.BigNumberRawValue{Value: fmt.Sprint(value), Precision: toInt(precision)}, totals.DefaultPrecision)
		if err != nil {
			return nil, err
		}
		rec.Set(p.Name, bn.Numeric())
	}
	return rec, nil
}

func toInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	}
	return 0
}

// serializeValue serializes a populated relation value.
func (r *Repository) serializeValue(value any, tree *fieldTree, prop *dml.MetaProperty, parents []string, idm identityMap) (any, error) {
	var joinFK []string
	if join, err := r.Schema.ResolveJoin(r.entityByClass(parents[len(parents)-1]), prop.Name); err == nil && join.PivotTable == "" && join.Many {
		joinFK = join.TargetColumns
	} else if err == nil && join.PivotTable == "" && prop.Kind == "1:1" && !prop.Owner {
		joinFK = join.TargetColumns
	}
	if tree != nil && tree.joined && !tree.all && prop.Kind == "1:m" {
		joinFK = nil
	}
	switch v := value.(type) {
	case *instance:
		if v == nil {
			return nil, nil
		}
		return r.serializeInstance(v, tree, parents, joinFK, idm)
	case []*instance:
		out := make([]*Record, 0, len(v))
		for _, c := range v {
			rec, err := r.serializeInstance(c, tree, parents, joinFK, idm)
			if err != nil {
				return nil, err
			}
			out = append(out, rec)
		}
		return out, nil
	default:
		return nil, nil
	}
}

func (r *Repository) entityByClass(className string) *dml.Entity {
	e, _ := r.Schema.Entity(className)
	return e
}
