package dal

import (
	"context"
	"fmt"
	"sort"
)

// The bigNumber properties of a DML entity keep their raw_<name> column in
// sync from the setter: hydrating an entity whose selection names a big
// number field but not its raw column (the JOINED load strategy selects
// only the fields asked for) sets raw_<name> in memory, so the unit of work
// finds the entity changed and updates it (raw_<name>, updated_at) when it
// flushes. markPartialBigNumbers records those updates for the relations
// loaded by a find inside a transaction.
func (r *Repository) markPartialBigNumbers(ctx context.Context, roots []*instance, tree *fieldTree) error {
	rec := recorderOf(r.DB)
	if rec == nil || tree == nil || !tree.joined || r.Schema == nil {
		return nil
	}
	var visit func(insts []*instance, node *fieldTree, isRoot bool) error
	visit = func(insts []*instance, node *fieldTree, isRoot bool) error {
		if len(insts) == 0 || node == nil {
			return nil
		}
		e := insts[0].entity
		if !isRoot && !node.all {
			var dirty []string
			for f := range bigNumberFields(e) {
				if node.fields[f] && !node.fields["raw_"+f] {
					dirty = append(dirty, f)
				}
			}
			sort.Strings(dirty)
			if len(dirty) > 0 {
				for _, inst := range insts {
					if err := r.updatePartialBigNumbers(ctx, rec, inst, dirty); err != nil {
						return err
					}
				}
			}
		}
		names := make([]string, 0, len(node.children))
		for name := range node.children {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			var children []*instance
			for _, inst := range insts {
				switch v := inst.relations[name].(type) {
				case *instance:
					children = append(children, v)
				case []*instance:
					children = append(children, v...)
				}
			}
			if err := visit(children, node.children[name], false); err != nil {
				return err
			}
		}
		return nil
	}
	return visit(roots, tree, true)
}

func (r *Repository) updatePartialBigNumbers(ctx context.Context, rec *Recorder, inst *instance, fields []string) error {
	e := inst.entity
	pkCol := fieldColumn(e, e.Meta.PrimaryKey[0])
	id := inst.data[pkCol]
	key := e.Meta.ClassName + "\x00" + fmt.Sprint(id)
	rec.mu.Lock()
	if rec.rawSynced == nil {
		rec.rawSynced = map[string]bool{}
	}
	done := rec.rawSynced[key]
	rec.rawSynced[key] = true
	rec.mu.Unlock()
	if done || id == nil {
		return nil
	}
	changed := false
	for _, f := range fields {
		if inst.data[f] != nil {
			changed = true
		}
	}
	if !changed {
		return nil
	}
	if _, ok := e.MetaProperty("updated_at"); ok {
		now := Now()
		sql := fmt.Sprintf("update %s set %s = $1 where %s = $2", quote(e.Meta.Collection), quote("updated_at"), quote(pkCol))
		if err := r.exec(ctx, sql, now, id); err != nil {
			return err
		}
		inst.data["updated_at"] = now
	}
	r.recordMutation(changeUpdate, ActionUpdated, e, inst.data)
	return nil
}
