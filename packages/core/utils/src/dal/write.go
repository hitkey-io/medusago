package dal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

// Now returns the current time with JavaScript Date precision.
var Now = func() time.Time { return time.Now().UTC().Truncate(time.Millisecond) }

var constraintPartsRe = regexp.MustCompile(`\([^(]*\)`)

func upperCaseFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// MapDBError mirrors dbErrorMapper from db-error-mapper.ts.
func MapDBError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	table := strings.Join(strings.Split(pgErr.TableName, "_"), " ")
	info := func() (keys, values []string, ok bool) {
		parts := constraintPartsRe.FindAllString(pgErr.Detail, -1)
		if len(parts) < 2 {
			return nil, nil, false
		}
		split := func(s string) []string {
			out := strings.Split(s[1:len(s)-1], ",")
			for i := range out {
				out[i] = strings.TrimSpace(out[i])
			}
			return out
		}
		return split(parts[0]), split(parts[1]), true
	}
	switch pgErr.Code {
	case "23505":
		keys, values, ok := info()
		if !ok {
			return err
		}
		pairs := make([]string, len(keys))
		for i, k := range keys {
			v := ""
			if i < len(values) {
				v = values[i]
			}
			switch v {
			case "t":
				v = "true"
			case "f":
				v = "false"
			}
			pairs[i] = k + ": " + v
		}
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA,
			fmt.Sprintf("%s with %s, already exists.", upperCaseFirst(table), strings.Join(pairs, ", ")))
	case "23502":
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA,
			fmt.Sprintf("Cannot set field '%s' of %s to null", pgErr.ColumnName, upperCaseFirst(table)))
	case "42703":
		msg := pgErr.Message
		if m := regexp.MustCompile(`(column.*)`).FindString(msg); m != "" {
			msg = m
		}
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, msg)
	case "23503":
		keys, values, _ := info()
		pairs := make([]string, len(keys))
		for i, k := range keys {
			v := ""
			if i < len(values) {
				v = values[i]
			}
			pairs[i] = k + ": " + v
		}
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND,
			fmt.Sprintf("You tried to set relationship %s, but such entity does not exist", strings.Join(pairs, ",")))
	}
	return err
}

func (r *Repository) exec(ctx context.Context, sql string, args ...any) error {
	_, err := r.DB.Exec(ctx, sql, args...)
	if err != nil {
		return MapDBError(err)
	}
	return nil
}

// columnValue converts a value for a column: arrays go to Postgres array
// columns as arrays, other structured values as JSON.
func columnValue(e *dml.Entity, column string, v any) (any, error) {
	if list, ok := v.([]any); ok {
		for _, p := range e.Meta.Properties {
			if len(p.FieldNames) == 0 || p.FieldNames[0] != column {
				continue
			}
			if strings.HasSuffix(p.ColumnType, "[]") || p.Type == "ArrayType" || p.Type == "string[]" {
				out := make([]string, len(list))
				for i, x := range list {
					out[i] = fmt.Sprint(x)
				}
				return out, nil
			}
		}
	}
	return sqlValue(v)
}

// sqlValue converts a record value into a query argument.
func sqlValue(v any) (any, error) {
	switch t := v.(type) {
	case *Record, []any:
		b, err := MarshalValue(t)
		return string(b), err
	case float64:
		return normalizeArg(t), nil
	default:
		return v, nil
	}
}

func decodeDefault(raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	return DecodeOrderedJSON(raw)
}

func parseDate(v any) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000Z", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC().Truncate(time.Millisecond)
		}
	}
	return v
}

// bigNumberColumns sets the numeric and raw columns of a BigNumber field.
func bigNumberColumns(data Row, name string, value any) error {
	if value == nil {
		data[name] = nil
		data["raw_"+name] = nil
		return nil
	}
	if raw, ok := value.(*Record); ok {
		v, _ := raw.Get("value")
		p, _ := raw.Get("precision")
		value = totals.BigNumberRawValue{Value: fmt.Sprint(v), Precision: toInt(p)}
	}
	bn, err := totals.NewBigNumber(value)
	if err != nil {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Cannot set value %v for %s.", value, name))
	}
	raw := bn.TrimmedRaw()
	rec := NewRecord()
	rec.Set("value", raw.Value)
	rec.Set("precision", float64(raw.Precision))
	data[name] = raw.Value
	data["raw_"+name] = rec
	return nil
}

func fieldColumn(e *dml.Entity, name string) string {
	if p, ok := e.MetaProperty(name); ok && len(p.FieldNames) > 0 {
		return p.FieldNames[0]
	}
	return name
}

// assignFields applies input fields to row data. With create set, missing
// fields get their DML default (or null when nullable).
func (r *Repository) assignFields(e *dml.Entity, data Row, input *Record, create bool) error {
	bigNumbers := bigNumberFields(e)
	for _, p := range e.Properties {
		f := p.Field
		if f == nil || f.Computed {
			continue
		}
		if strings.HasPrefix(p.Name, "raw_") && bigNumbers[strings.TrimPrefix(p.Name, "raw_")] {
			continue
		}
		value, provided := input.Get(p.Name)
		if !provided {
			if !create {
				continue
			}
			switch {
			case f.DataType.Name == "id" && f.DataType.Options != nil:
				value = common.GenerateEntityID("", f.DataType.Options.Prefix)
			case len(f.DefaultValue) > 0:
				d, err := decodeDefault(f.DefaultValue)
				if err != nil {
					return err
				}
				value = d
			case f.Nullable:
				value = nil
			default:
				continue
			}
		}
		if f.DataType.Name == "id" && value == "" && create && f.DataType.Options != nil {
			value = common.GenerateEntityID("", f.DataType.Options.Prefix)
		}
		switch f.DataType.Name {
		case "bigNumber":
			if err := bigNumberColumns(data, fieldColumn(e, p.Name), value); err != nil {
				return err
			}
		case "dateTime":
			data[fieldColumn(e, p.Name)] = parseDate(value)
		default:
			data[fieldColumn(e, p.Name)] = value
		}
	}

	// Foreign keys, given either as <relation>_id or as the relation.
	// The FK is either a mapToPk relation property or (pivot models such as
	// ProductProductOption) a plain scalar sharing its column with a
	// non-persisted relation.
	dmlFields := map[string]bool{}
	for _, p := range e.Properties {
		if p.Field != nil {
			dmlFields[p.Name] = true
		}
	}
	isToOne := func(k string) bool { return k == "m:1" || k == "1:1" }
	for _, mp := range e.Meta.Properties {
		if (mp.Persist != nil && !*mp.Persist) || len(mp.FieldNames) == 0 {
			continue
		}
		if !mp.MapToPk {
			if mp.Kind != "scalar" || dmlFields[mp.Name] {
				continue
			}
			shared := false
			for _, rel := range e.Meta.Properties {
				if isToOne(rel.Kind) && len(rel.FieldNames) > 0 && rel.FieldNames[0] == mp.FieldNames[0] {
					shared = true
					break
				}
			}
			if !shared {
				continue
			}
		}
		if v, ok := input.Get(mp.Name); ok {
			data[mp.FieldNames[0]] = v
			continue
		}
		for _, rel := range e.Meta.Properties {
			if !isToOne(rel.Kind) || rel.MapToPk || rel.Name == mp.Name || len(rel.FieldNames) == 0 || rel.FieldNames[0] != mp.FieldNames[0] {
				continue
			}
			v, ok := input.Get(rel.Name)
			if !ok {
				break
			}
			if obj, isObj := v.(*Record); isObj {
				v, _ = obj.Get("id")
			}
			data[mp.FieldNames[0]] = v
		}
		if _, set := data[mp.FieldNames[0]]; !set && create {
			if mp.Nullable {
				data[mp.FieldNames[0]] = nil
			}
		}
	}

	now := Now()
	if create {
		if _, ok := e.MetaProperty("created_at"); ok {
			if v, given := input.Get("created_at"); given {
				data["created_at"] = parseDate(v)
			} else {
				data["created_at"] = now
			}
		}
		if _, ok := e.MetaProperty("updated_at"); ok {
			if v, given := input.Get("updated_at"); given {
				data["updated_at"] = parseDate(v)
			} else {
				data["updated_at"] = now
			}
		}
		if _, ok := e.MetaProperty("deleted_at"); ok {
			if v, given := input.Get("deleted_at"); given {
				data["deleted_at"] = parseDate(v)
			} else {
				data["deleted_at"] = nil
			}
		}
	}
	return nil
}

func (r *Repository) insertRow(ctx context.Context, e *dml.Entity, data Row) error {
	var cols, ph []string
	var args []any
	for _, mp := range e.Meta.Properties {
		if len(mp.FieldNames) == 0 || (mp.Persist != nil && !*mp.Persist) || mp.Kind == "1:m" || mp.Kind == "m:n" {
			continue
		}
		if mp.Kind == "1:1" && !mp.Owner {
			continue
		}
		column := mp.FieldNames[0]
		v, ok := data[column]
		if !ok || slices.Contains(cols, quote(column)) {
			continue
		}
		arg, err := columnValue(e, column, v)
		if err != nil {
			return err
		}
		args = append(args, arg)
		cols = append(cols, quote(column))
		ph = append(ph, fmt.Sprintf("$%d", len(args)))
	}
	sql := fmt.Sprintf("insert into %s (%s) values (%s)", quote(e.Meta.Collection), strings.Join(cols, ", "), strings.Join(ph, ", "))
	return r.exec(ctx, sql, args...)
}

// Create mirrors repository.create + flush: every input becomes an entity,
// with nested one-to-many children created and many-to-many items attached.
func (r *Repository) Create(ctx context.Context, inputs []*Record) ([]*Record, error) {
	idm := identityMap{}
	var out []*Record
	for _, input := range inputs {
		inst, err := r.create(ctx, r.Entity, input, idm)
		if err != nil {
			return nil, err
		}
		rec, err := r.serializeCreated(inst, idm)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

func (r *Repository) create(ctx context.Context, e *dml.Entity, input *Record, idm identityMap) (*instance, error) {
	data := Row{}
	if err := r.assignFields(e, data, input, true); err != nil {
		return nil, err
	}
	if err := r.insertRow(ctx, e, data); err != nil {
		return nil, err
	}
	inst := idm.get(e, data)
	inst.data = data
	inst.created = true

	for _, mp := range e.Meta.Properties {
		if mp.Kind != "1:m" && mp.Kind != "m:n" {
			continue
		}
		inst.relations[mp.Name] = []*instance{}
		value, ok := input.Get(mp.Name)
		if !ok {
			continue
		}
		items, _ := value.([]any)
		join, err := r.Schema.ResolveJoin(e, mp.Name)
		if err != nil {
			return nil, err
		}
		var children []*instance
		for _, item := range items {
			child, err := r.attachChild(ctx, inst, join, item, idm)
			if err != nil {
				return nil, err
			}
			children = append(children, child)
		}
		inst.relations[mp.Name] = children
	}
	return inst, nil
}

func (r *Repository) attachChild(ctx context.Context, parent *instance, join *Join, item any, idm identityMap) (*instance, error) {
	target := join.Target
	obj, isObj := item.(*Record)
	var id any
	if isObj {
		id, _ = obj.Get(target.Meta.PrimaryKey[0])
	} else {
		id = item
	}

	if join.PivotTable == "" {
		// one-to-many: create the child with the foreign key set.
		child := NewRecord()
		if isObj {
			for _, k := range obj.Keys() {
				v, _ := obj.Get(k)
				child.Set(k, v)
			}
		}
		inverse, _ := target.MetaProperty(join.Relation.MappedBy)
		for _, mp := range target.Meta.Properties {
			if mp.MapToPk && len(mp.FieldNames) > 0 && inverse != nil && len(inverse.FieldNames) > 0 && mp.FieldNames[0] == inverse.FieldNames[0] {
				child.Set(mp.Name, parent.data[join.SourceColumns[0]])
			}
		}
		return r.create(ctx, target, child, idm)
	}

	// many-to-many: create the item when needed, then link it.
	var child *instance
	if id == nil {
		created, err := r.create(ctx, target, obj, idm)
		if err != nil {
			return nil, err
		}
		child = created
		id = created.data[join.TargetColumns[0]]
	} else {
		child = idm.get(target, Row{target.Meta.PrimaryKey[0]: id})
	}
	sql := fmt.Sprintf("insert into %s (%s, %s) values ($1, $2)", quote(join.PivotTable), quote(join.PivotSourceColumns[0]), quote(join.PivotTargetColumns[0]))
	if pivot, err := r.Schema.Entity(join.Relation.PivotEntity); err == nil && join.Relation.PivotEntity != "" {
		data := Row{}
		input := NewRecord()
		input.Set(join.PivotSourceColumns[0], parent.data[join.SourceColumns[0]])
		input.Set(join.PivotTargetColumns[0], id)
		if err := r.assignFields(pivot, data, input, true); err != nil {
			return nil, err
		}
		data[join.PivotSourceColumns[0]] = parent.data[join.SourceColumns[0]]
		data[join.PivotTargetColumns[0]] = id
		return child, r.insertRow(ctx, pivot, data)
	}
	return child, r.exec(ctx, sql, parent.data[join.SourceColumns[0]], id)
}

// serializeCreated serializes a newly created entity: every field, foreign
// keys and the initialized collections, without unloaded references.
func (r *Repository) serializeCreated(inst *instance, idm identityMap) (*Record, error) {
	tree := &fieldTree{all: true, fields: map[string]bool{}, children: map[string]*fieldTree{}}
	for name, v := range inst.relations {
		if _, ok := v.([]*instance); ok {
			tree.children[name] = allFields
		}
	}
	return r.serializeInstance(inst, tree, nil, nil, idm)
}

// UpdateInput is one entity and the data assigned to it.
type UpdateInput struct {
	Record *Record
	Data   *Record
}

// UpdateWith mirrors repository.update: the entities matching opts are
// loaded, the data returned by dataFor is assigned to each of them and the
// changed columns are written, refreshing updated_at.
func (r *Repository) UpdateWith(ctx context.Context, opts FindOptions, dataFor func(current Row) *Record) ([]*Record, error) {
	roots, idm, err := r.find(ctx, opts)
	if err != nil {
		return nil, err
	}
	for _, inst := range roots {
		data := dataFor(inst.data)
		if data == nil {
			continue
		}
		if err := r.applyUpdate(ctx, inst, data); err != nil {
			return nil, err
		}
	}
	return r.serialize(roots, idm, opts)
}

// applyUpdate assigns data to a loaded entity and writes the changed
// columns (MikroORM only flushes dirty entities, refreshing updated_at).
func (r *Repository) applyUpdate(ctx context.Context, inst *instance, data *Record) error {
	e := inst.entity
	changed := Row{}
	if err := r.assignFields(e, changed, data, false); err != nil {
		return err
	}
	for column, v := range changed {
		if sameValue(inst.data[column], v) {
			delete(changed, column)
		}
	}
	if len(changed) == 0 {
		return nil
	}
	if _, ok := e.MetaProperty("updated_at"); ok {
		changed["updated_at"] = Now()
	}
	columns := make([]string, 0, len(changed))
	for c := range changed {
		columns = append(columns, c)
	}
	slices.Sort(columns)
	var sets []string
	var args []any
	for _, c := range columns {
		arg, err := columnValue(e, c, changed[c])
		if err != nil {
			return err
		}
		args = append(args, arg)
		sets = append(sets, fmt.Sprintf("%s = $%d", quote(c), len(args)))
		inst.data[c] = changed[c]
	}
	var where []string
	for _, pk := range e.Meta.PrimaryKey {
		args = append(args, inst.data[pk])
		where = append(where, fmt.Sprintf("%s = $%d", quote(fieldColumn(e, pk)), len(args)))
	}
	sql := fmt.Sprintf("update %s set %s where %s", quote(e.Meta.Collection), strings.Join(sets, ", "), strings.Join(where, " and "))
	return r.exec(ctx, sql, args...)
}

// Upsert mirrors repository.upsert: existing entities (by primary key, not
// soft deleted) are updated, the others created; created entities are
// returned first.
func (r *Repository) Upsert(ctx context.Context, data []*Record) ([]*Record, error) {
	pks := r.Entity.Meta.PrimaryKey
	if len(pks) != 1 {
		return nil, fmt.Errorf("upsert of %s: composite primary keys are not supported yet", r.Entity.Meta.ClassName)
	}
	pk := pks[0]
	var values []any
	for _, d := range data {
		if v, ok := d.Get(pk); ok && v != nil && v != "" {
			values = append(values, v)
		}
	}
	existing := map[string]*instance{}
	idm := identityMap{}
	if len(values) > 0 {
		where := NewRecord()
		where.Set(pk, values)
		roots, found, err := r.find(ctx, FindOptions{Where: where})
		if err != nil {
			return nil, err
		}
		idm = found
		for _, inst := range roots {
			existing[fmt.Sprint(inst.data[fieldColumn(r.Entity, pk)])] = inst
		}
	}

	var created, updated []*instance
	for _, d := range data {
		v, _ := d.Get(pk)
		if inst, ok := existing[fmt.Sprint(v)]; ok && v != nil {
			if err := r.applyUpdate(ctx, inst, d); err != nil {
				return nil, err
			}
			updated = append(updated, inst)
			continue
		}
		inst, err := r.create(ctx, r.Entity, d, idm)
		if err != nil {
			return nil, err
		}
		created = append(created, inst)
	}

	out := make([]*Record, 0, len(created)+len(updated))
	for _, inst := range created {
		rec, err := r.serializeCreated(inst, idm)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if len(updated) > 0 {
		recs, err := r.serialize(updated, idm, FindOptions{})
		if err != nil {
			return nil, err
		}
		out = append(out, recs...)
	}
	return out, nil
}

func sameValue(a, b any) bool {
	ja, errA := MarshalValue(a)
	jb, errB := MarshalValue(b)
	if errA != nil || errB != nil {
		return false
	}
	// numeric columns are read as strings; compare numerically.
	return string(ja) == string(jb) || strings.Trim(string(ja), `"`) == strings.Trim(string(jb), `"`)
}

// Delete mirrors repository.delete: rows matching the filter are deleted
// (soft deleted ones included) and their primary keys returned.
func (r *Repository) Delete(ctx context.Context, where *Record) ([]any, error) {
	qb := newQueryBuilder(r.Schema, true)
	cond, err := qb.where(r.Entity, "e0", where, "e0")
	if err != nil {
		return nil, err
	}
	if cond == "" {
		cond = "true"
	}
	pk := r.Entity.Meta.PrimaryKey[0]
	sql := fmt.Sprintf(`delete from %s as "e0" where %s returning %s`, quote(r.Entity.Meta.Collection), cond, col("e0", fieldColumn(r.Entity, pk)))
	rows, err := queryRows(ctx, r.DB, sql, qb.args...)
	if err != nil {
		return nil, MapDBError(err)
	}
	ids := make([]any, len(rows))
	for i, row := range rows {
		ids[i] = row[fieldColumn(r.Entity, pk)]
	}
	return ids, nil
}

// CascadeResult lists the soft deleted (or restored) entities by class, in
// the order they were reached.
type CascadeResult struct {
	Classes  []string
	Entities map[string][]Row
}

func (c *CascadeResult) add(class string, row Row) {
	if c.Entities == nil {
		c.Entities = map[string][]Row{}
	}
	if _, ok := c.Entities[class]; !ok {
		c.Classes = append(c.Classes, class)
	}
	c.Entities[class] = append(c.Entities[class], row)
}

// SoftDelete mirrors repository.softDelete with
// mikroOrmUpdateDeletedAtRecursively: the matching entities and, through the
// relations cascaded on delete, their children (already deleted ones
// included) get deleted_at set.
func (r *Repository) SoftDelete(ctx context.Context, where *Record) (*CascadeResult, error) {
	return r.updateDeletedAt(ctx, where, false, Now())
}

// Restore mirrors repository.restore.
func (r *Repository) Restore(ctx context.Context, where *Record) (*CascadeResult, error) {
	return r.updateDeletedAt(ctx, where, true, time.Time{})
}

func (r *Repository) updateDeletedAt(ctx context.Context, where *Record, withDeleted bool, value time.Time) (*CascadeResult, error) {
	roots, idm, err := r.find(ctx, FindOptions{Where: where, WithDeleted: withDeleted})
	if err != nil {
		return nil, err
	}
	result := &CascadeResult{}
	var deletedAt any
	if !value.IsZero() {
		deletedAt = value
	}

	type pending struct {
		entity *dml.Entity
		insts  []*instance
	}
	level := []pending{{r.Entity, roots}}
	touched := map[*dml.Entity][]*instance{}
	var order []*dml.Entity
	for len(level) > 0 {
		var next []pending
		for _, group := range level {
			if !hasDeletedAt(group.entity) {
				continue
			}
			if _, ok := touched[group.entity]; !ok {
				order = append(order, group.entity)
			}
			for _, inst := range group.insts {
				inst.data["deleted_at"] = deletedAt
				result.add(group.entity.Meta.ClassName, inst.data)
				touched[group.entity] = append(touched[group.entity], inst)
			}
			for _, relation := range group.entity.Cascades.Delete {
				children, err := r.loadRelation(ctx, group.insts, relation, idm, true, nil)
				if err != nil {
					return nil, err
				}
				if len(children) == 0 {
					continue
				}
				next = append(next, pending{children[0].entity, children})
			}
		}
		level = next
	}

	now := Now()
	for _, e := range order {
		insts := touched[e]
		if len(insts) == 0 {
			continue
		}
		var args []any
		args = append(args, deletedAt, now)
		ph := make([]string, len(insts))
		for i, inst := range insts {
			args = append(args, inst.data[e.Meta.PrimaryKey[0]])
			ph[i] = fmt.Sprintf("$%d", len(args))
			inst.data["updated_at"] = now
		}
		sql := fmt.Sprintf(`update %s set "deleted_at" = $1, "updated_at" = $2 where %s in (%s)`,
			quote(e.Meta.Collection), quote(fieldColumn(e, e.Meta.PrimaryKey[0])), strings.Join(ph, ", "))
		if err := r.exec(ctx, sql, args...); err != nil {
			return nil, err
		}
	}
	return result, nil
}
