package dal

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
)

// newEntityID mirrors the id MikroORM assigns in getEntityWithId (the id
// property's prefix).
func newEntityID(e *dml.Entity) any {
	for _, p := range e.Properties {
		if p.Field != nil && p.Field.DataType.Name == "id" && p.Field.DataType.Options != nil {
			return common.GenerateEntityID("", p.Field.DataType.Options.Prefix)
		}
	}
	return common.GenerateEntityID("", "")
}

func withEntityID(e *dml.Entity, data *Record) *Record {
	out := NewRecord()
	for _, k := range data.Keys() {
		v, _ := data.Get(k)
		out.Set(k, v)
	}
	// { ...data, ...created.__helper.__bignumberdata, id }: the raw values
	// of the BigNumber fields given
	bigNumbers := bigNumberFields(e)
	for _, k := range data.Keys() {
		v, _ := data.Get(k)
		if !bigNumbers[k] || v == nil {
			continue
		}
		row := Row{}
		if err := bigNumberColumns(row, k, v); err == nil {
			out.Set("raw_"+k, row["raw_"+k])
		}
	}
	if id, ok := out.Get("id"); !ok || id == nil || id == "" {
		out.Set("id", newEntityID(e))
	}
	return out
}

// UpsertWithReplace mirrors repository.upsertWithReplace(data, { relations }):
// top-level entities are inserted or natively updated (no dirty check,
// updated_at refreshed), then the given collection relations are replaced:
// one-to-many children missing from the input are hard deleted and the
// others upserted; many-to-many pivots are replaced.
func (r *Repository) UpsertWithReplace(ctx context.Context, data []*Record, relations []string) ([]*Record, error) {
	if len(data) == 0 {
		return []*Record{}, nil
	}
	e := r.Entity
	var relationProps []*dml.MetaProperty
	for i := range e.Meta.Properties {
		mp := &e.Meta.Properties[i]
		if mp.Kind != "scalar" {
			relationProps = append(relationProps, mp)
		}
	}
	var missing []string
	for _, name := range relations {
		found := false
		for _, mp := range relationProps {
			if mp.Name == name {
				found = true
			}
		}
		if !found {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Nonexistent relations were passed during upsert: "+strings.Join(missing, ","))
	}

	mains := make([]*Record, len(data))
	response := make([]*Record, len(data))
	for i, entry := range data {
		copyRec := NewRecord()
		relValues := NewRecord()
		for _, k := range entry.Keys() {
			v, _ := entry.Get(k)
			copyRec.Set(k, v)
		}
		for _, mp := range relationProps {
			v, ok := copyRec.Get(mp.Name)
			if !ok {
				continue
			}
			copyRec.Delete(mp.Name)
			joinColumns := mp.JoinColumns
			if len(joinColumns) == 0 {
				joinColumns = mp.FieldNames
			}
			if mp.Kind == "m:1" && len(joinColumns) > 0 {
				switch t := v.(type) {
				case nil:
					copyRec.Set(joinColumns[0], nil)
				case string:
					copyRec.Set(joinColumns[0], t)
				case *Record:
					if id, ok := t.Get("id"); ok {
						copyRec.Set(joinColumns[0], id)
					}
				}
			}
			relValues.Set(mp.Name, v)
		}
		main := withEntityID(e, copyRec)
		mains[i] = main
		resp := NewRecord()
		for _, k := range main.Keys() {
			v, _ := main.Get(k)
			resp.Set(k, v)
		}
		for _, k := range relValues.Keys() {
			v, _ := relValues.Get(k)
			resp.Set(k, v)
		}
		response[i] = resp
		// keep the relation input for the collection pass
		for _, k := range relValues.Keys() {
			v, _ := relValues.Get(k)
			if _, ok := entry.Get(k); ok {
				mains[i].Set("\x00"+k, v)
			}
		}
	}
	plain := make([]*Record, len(mains))
	for i, m := range mains {
		p := NewRecord()
		for _, k := range m.Keys() {
			if strings.HasPrefix(k, "\x00") {
				continue
			}
			v, _ := m.Get(k)
			p.Set(k, v)
		}
		plain[i] = p
	}
	pa := &performedActions{}
	if err := r.upsertMany(ctx, e, plain, false, pa); err != nil {
		return nil, err
	}

	for _, name := range relations {
		mp, _ := e.MetaProperty(name)
		if mp.Kind != "1:m" && mp.Kind != "m:n" {
			continue
		}
		join, err := r.Schema.ResolveJoin(e, name)
		if err != nil {
			return nil, err
		}
		type parentItems struct {
			index int
			id    any
			items []*Record
		}
		var parents []parentItems
		for i, m := range mains {
			v, ok := m.Get("\x00" + name)
			if !ok {
				continue
			}
			var items []*Record
			switch t := v.(type) {
			case []any:
				for _, x := range t {
					if rec, ok := x.(*Record); ok {
						items = append(items, withEntityID(join.Target, rec))
					}
				}
			case []*Record:
				for _, rec := range t {
					items = append(items, withEntityID(join.Target, rec))
				}
			}
			id, _ := m.Get("id")
			parents = append(parents, parentItems{index: i, id: id, items: items})
		}
		if len(parents) == 0 {
			continue
		}
		if mp.Kind == "1:m" {
			var all []*Record
			var parentIDs []any
			var keepIDs []string
			for _, p := range parents {
				parentIDs = append(parentIDs, p.id)
				for _, it := range p.items {
					it.Set(join.TargetColumns[0], p.id)
					all = append(all, it)
					id, _ := it.Get("id")
					keepIDs = append(keepIDs, fmt.Sprint(id))
				}
				list := make([]any, len(p.items))
				for j, it := range p.items {
					list[j] = it
				}
				response[p.index].Set(name, list)
			}
			sql := fmt.Sprintf("delete from %s where %s = any($1)", quote(join.Target.Meta.Collection), quote(join.TargetColumns[0]))
			args := []any{toStrings(parentIDs)}
			if len(keepIDs) > 0 {
				sql += " and \"id\" <> all($2)"
				args = append(args, keepIDs)
			}
			deletedRows, err := QueryRows(ctx, r.DB, sql+` returning "id"`, args...)
			if err != nil {
				return nil, MapDBError(err)
			}
			for _, row := range deletedRows {
				pa.deleted.add(join.Target, row["id"])
			}
			if err := r.upsertMany(ctx, join.Target, all, false, pa); err != nil {
				return nil, err
			}
			continue
		}
		// many-to-many
		for _, p := range parents {
			list := make([]any, len(p.items))
			for j, it := range p.items {
				list[j] = it
			}
			response[p.index].Set(name, list)
			if len(p.items) > 0 {
				if err := r.upsertMany(ctx, join.Target, p.items, true, pa); err != nil {
					return nil, err
				}
			}
			ids := make([]any, len(p.items))
			for j, it := range p.items {
				ids[j], _ = it.Get("id")
			}
			if err := r.replaceManyToMany(ctx, p.id, name, ids, pa); err != nil {
				return nil, err
			}
		}
	}
	r.dispatchPerformedActions(pa)
	return response, nil
}

func toStrings(values []any) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = fmt.Sprint(v)
	}
	return out
}

// upsertMany mirrors upsertMany_: rows whose id exists are natively updated
// (unless skipUpdate), the others inserted.
func (r *Repository) upsertMany(ctx context.Context, e *dml.Entity, entries []*Record, skipUpdate bool, pa *performedActions) error {
	if len(entries) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var unique []*Record
	var ids []string
	for _, en := range entries {
		id, _ := en.Get("id")
		k := fmt.Sprint(id)
		if seen[k] {
			continue
		}
		seen[k] = true
		unique = append(unique, en)
		ids = append(ids, k)
	}
	rows, err := QueryRows(ctx, r.DB, fmt.Sprintf("select \"id\" from %s where \"id\" = any($1)", quote(e.Meta.Collection)), ids)
	if err != nil {
		return MapDBError(err)
	}
	existing := map[string]bool{}
	for _, row := range rows {
		existing[fmt.Sprint(row["id"])] = true
	}
	var created, updated []any
	for _, en := range unique {
		id, _ := en.Get("id")
		if existing[fmt.Sprint(id)] {
			if skipUpdate {
				continue
			}
			if err := r.nativeUpdate(ctx, e, en); err != nil {
				return err
			}
			updated = append(updated, id)
			continue
		}
		data := Row{}
		if err := r.assignFields(e, data, en, true); err != nil {
			return err
		}
		if err := r.insertRow(ctx, e, data); err != nil {
			return err
		}
		created = append(created, data[fieldColumn(e, "id")])
	}
	if pa != nil {
		pa.created.add(e, created...)
		pa.updated.add(e, updated...)
	}
	return nil
}

// nativeUpdate mirrors nativeUpdateMany for one row: the given columns and
// the onUpdate properties (updated_at).
func (r *Repository) nativeUpdate(ctx context.Context, e *dml.Entity, en *Record) error {
	changed := Row{}
	if err := r.assignFields(e, changed, en, false); err != nil {
		return err
	}
	delete(changed, "id")
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
	}
	if len(sets) == 0 {
		return nil
	}
	id, _ := en.Get("id")
	args = append(args, id)
	return r.exec(ctx, fmt.Sprintf("update %s set %s where \"id\" = $%d", quote(e.Meta.Collection), strings.Join(sets, ", "), len(args)), args...)
}
