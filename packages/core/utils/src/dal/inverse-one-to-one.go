package dal

import (
	"context"
	"fmt"
)

// inverseRefKey is the instance data key holding the primary key of the
// owner row of an inverse 1:1 relation.
func inverseRefKey(rel string) string { return "\x00ref:" + rel }

// loadInverseOneToOneRefs mirrors MikroORM's autoJoinOneToOneOwner: for the
// inverse side of a 1:1 relation the owner's primary key is loaded with the
// entity, so an unpopulated relation serializes as a reference ({ id }).
func (r *Repository) loadInverseOneToOneRefs(ctx context.Context, idm identityMap) error {
	type group struct {
		insts []*instance
	}
	byClass := map[string][]*instance{}
	for _, inst := range idm {
		byClass[inst.entity.Meta.ClassName] = append(byClass[inst.entity.Meta.ClassName], inst)
	}
	for _, insts := range byClass {
		e := insts[0].entity
		for i := range e.Meta.Properties {
			p := &e.Meta.Properties[i]
			if p.Kind != "1:1" || p.Owner || p.MappedBy == "" {
				continue
			}
			target, err := r.Schema.Entity(p.Entity)
			if err != nil {
				continue
			}
			owner, ok := target.MetaProperty(p.MappedBy)
			if !ok || len(owner.FieldNames) == 0 {
				continue
			}
			var keys []string
			seen := map[string]bool{}
			for _, inst := range insts {
				if _, populated := inst.relations[p.Name]; populated {
					continue
				}
				if _, done := inst.data[inverseRefKey(p.Name)]; done {
					continue
				}
				k := fmt.Sprint(inst.data[e.Meta.PrimaryKey[0]])
				if !seen[k] {
					seen[k] = true
					keys = append(keys, k)
				}
			}
			if len(keys) == 0 {
				continue
			}
			cond := ""
			if hasDeletedAt(target) {
				cond = ` and "deleted_at" is null`
			}
			sql := fmt.Sprintf(`select %s as "pk", %s as "fk" from %s where %s = any($1)%s`,
				quote(fieldColumn(target, target.Meta.PrimaryKey[0])), quote(owner.FieldNames[0]), quote(target.Meta.Collection), quote(owner.FieldNames[0]), cond)
			rows, err := QueryRows(ctx, r.DB, sql, keys)
			if err != nil {
				return MapDBError(err)
			}
			refs := map[string]any{}
			for _, row := range rows {
				refs[fmt.Sprint(row["fk"])] = row["pk"]
			}
			for _, inst := range insts {
				if _, populated := inst.relations[p.Name]; populated {
					continue
				}
				inst.data[inverseRefKey(p.Name)] = refs[fmt.Sprint(inst.data[e.Meta.PrimaryKey[0]])]
			}
		}
	}
	return nil
}
