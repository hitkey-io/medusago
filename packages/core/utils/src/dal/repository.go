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
		return inst
	}
	inst := &instance{entity: e, data: row, relations: map[string]any{}}
	m[key] = inst
	return inst
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
	roots, idm, err := r.find(ctx, opts)
	if err != nil {
		return nil, err
	}
	return r.serialize(roots, idm, opts)
}

// FindAndCount mirrors repository.findAndCount.
func (r *Repository) FindAndCount(ctx context.Context, opts FindOptions) ([]*Record, int, error) {
	records, err := r.Find(ctx, opts)
	if err != nil {
		return nil, 0, err
	}
	count, err := r.count(ctx, opts)
	return records, count, err
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
	qb := newQueryBuilder(r.Schema, opts.WithDeleted)
	where, err := qb.rootWhere(r.Entity, "e0", opts.Where)
	if err != nil {
		return nil, nil, err
	}
	order, err := r.orderSQL(opts.Order, "e0")
	if err != nil {
		return nil, nil, err
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
		sql = fmt.Sprintf(`select "e0".* from %s as "e0"%s%s%s%s`, table, qb.joinSQL(), where, order, paging)
	}

	rows, err := queryRows(ctx, r.DB, sql, qb.args...)
	if err != nil {
		return nil, nil, err
	}

	idm := identityMap{}
	var roots []*instance
	seen := map[*instance]bool{}
	for _, row := range rows {
		inst := idm.get(r.Entity, row)
		if !seen[inst] {
			seen[inst] = true
			roots = append(roots, inst)
		}
	}

	if err := r.populate(ctx, roots, idm, r.relationPaths(opts), opts.WithDeleted); err != nil {
		return nil, nil, err
	}
	return roots, idm, nil
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
func (r *Repository) populate(ctx context.Context, roots []*instance, idm identityMap, paths []string, withDeleted bool) error {
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
		children, err := r.loadRelation(ctx, parents, relation, idm, withDeleted)
		if err != nil {
			return fmt.Errorf("populate %s: %w", path, err)
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

func (r *Repository) loadRelation(ctx context.Context, parents []*instance, relation string, idm identityMap, withDeleted bool) ([]*instance, error) {
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
		sql := fmt.Sprintf(`select "p0".%s as "__pivot_source", "t1".* from %s as "p0" inner join %s as "t1" on "p0".%s = "t1".%s where "p0".%s in (%s)%s%s`,
			quote(join.PivotSourceColumns[0]), quote(join.PivotTable), quote(target.Meta.Collection),
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
		where := fmt.Sprintf(`"t0".%s in (%s)`, quote(join.TargetColumns[0]), placeholders(values))
		if softDelete {
			where = `"t0"."deleted_at" is null and ` + where
		}
		sql := fmt.Sprintf(`select "t0".* from %s as "t0" where %s`, quote(target.Meta.Collection), where)
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

// fieldTree is the selection for one entity level.
type fieldTree struct {
	all      bool
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

	for _, pk := range e.Meta.PrimaryKey {
		rec.Set(pk, inst.data[pk])
	}

	var collections []*dml.MetaProperty
	for i := range e.Meta.Properties {
		p := &e.Meta.Properties[i]
		switch {
		case p.Kind == "scalar":
			if bigNumbers[p.Name] || !hydrated(p.Name) {
				continue
			}
			rec.Set(p.Name, inst.data[p.FieldNames[0]])

		case p.MapToPk && (p.Persist == nil || *p.Persist):
			// Foreign key property (e.g. product_id).
			if hydrated(p.Name) || requiredFK[p.FieldNames[0]] {
				rec.Set(p.Name, inst.data[p.FieldNames[0]])
			}

		case p.Kind == "m:1" || p.Kind == "1:1":
			if isParent(p) {
				continue
			}
			if child, populated := tree.children[p.Name]; populated {
				v, err := r.serializeValue(inst.relations[p.Name], child, p, childParents, idm)
				if err != nil {
					return nil, err
				}
				rec.Set(p.Name, v)
				continue
			}
			if len(p.FieldNames) == 0 || !(hydrated(p.Name) || hydrated(p.FieldNames[0])) {
				continue
			}
			fk := inst.data[p.FieldNames[0]]
			if fk == nil {
				rec.Set(p.Name, nil)
				continue
			}
			target, err := r.Schema.Entity(p.Entity)
			if err != nil {
				return nil, err
			}
			if loaded, ok := idm.lookup(target, fk); ok {
				ref, err := r.serializeInstance(loaded, allFields, childParents, nil, idm)
				if err != nil {
					return nil, err
				}
				rec.Set(p.Name, ref)
			} else {
				ref := NewRecord()
				ref.Set(target.Meta.PrimaryKey[0], fk)
				rec.Set(p.Name, ref)
			}

		default: // 1:m, m:n
			if _, populated := tree.children[p.Name]; populated && !isParent(p) {
				collections = append(collections, p)
			}
		}
	}

	for _, p := range collections {
		v, err := r.serializeValue(inst.relations[p.Name], tree.children[p.Name], p, childParents, idm)
		if err != nil {
			return nil, err
		}
		rec.Set(p.Name, v)
	}

	for _, p := range e.Properties {
		if !bigNumbers[p.Name] || !hydrated(p.Name) {
			continue
		}
		raw, ok := inst.data["raw_"+p.Name].(*Record)
		if !ok {
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
