package dal

import (
	"fmt"
	"maps"
	"strconv"
	"sync"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
)

// OnInitHook mirrors a MikroORM @OnInit hook: it runs when an entity is
// hydrated and can set computed values in its row.
type OnInitHook func(data Row)

// OnInitRefHook is an @OnInit hook that reads related entities: ref returns
// the data of the entity a to-one relation points to, as MikroORM sees it
// while hydrating: the managed entity's loaded columns when it is in the
// identity map, an empty row for an unloaded reference, nil without a
// foreign key.
type OnInitRefHook func(data Row, ref func(relation string) Row)

var (
	hooksMu        sync.RWMutex
	onInitHooks    = map[string][]OnInitHook{}
	onInitRefHooks = map[string][]OnInitRefHook{}
)

// RegisterOnInitWithRefs registers an @OnInit hook reading related entities.
func RegisterOnInitWithRefs(className string, hook OnInitRefHook) {
	hooksMu.Lock()
	defer hooksMu.Unlock()
	onInitRefHooks[className] = append(onInitRefHooks[className], hook)
}

// initEntity runs the @OnInit hooks of a hydrated entity. Columns the hooks
// change are remembered (with their hydrated value): MikroORM's original
// entity data predates the hooks, so a later flush writes them.
func (m identityMap) initEntity(inst *instance) {
	className := inst.entity.Meta.ClassName
	hooksMu.RLock()
	refHooks := onInitRefHooks[className]
	hooksMu.RUnlock()
	if len(refHooks) == 0 {
		runOnInit(className, inst.data)
		return
	}
	before := maps.Clone(inst.data)
	runOnInit(className, inst.data)
	e := inst.entity
	ref := func(relation string) Row {
		p, ok := e.MetaProperty(relation)
		if !ok || len(p.FieldNames) == 0 {
			return nil
		}
		fk := inst.data[p.FieldNames[0]]
		if fk == nil {
			return nil
		}
		if target, ok := m[p.Entity+"\x00"+fmt.Sprint(fk)]; ok {
			return target.data
		}
		return Row{}
	}
	for _, h := range refHooks {
		h(inst.data, ref)
	}
	for k, v := range inst.data {
		old, had := before[k]
		if !had || !sameValue(old, v) {
			if inst.initChanged == nil {
				inst.initChanged = map[string]bool{}
			}
			if isPersistedColumn(e, k) {
				inst.initChanged[k] = true
			}
		}
	}
}

func isPersistedColumn(e *dml.Entity, column string) bool {
	for _, p := range e.Meta.Properties {
		if len(p.FieldNames) > 0 && p.FieldNames[0] == column && (p.Persist == nil || *p.Persist) && p.Kind == "scalar" {
			return true
		}
	}
	return false
}

// RegisterOnInit registers an @OnInit hook for an entity class.
func RegisterOnInit(className string, hook OnInitHook) {
	hooksMu.Lock()
	defer hooksMu.Unlock()
	onInitHooks[className] = append(onInitHooks[className], hook)
}

func runOnInit(className string, data Row) {
	hooksMu.RLock()
	hooks := onInitHooks[className]
	hooksMu.RUnlock()
	for _, h := range hooks {
		h(data)
	}
}

// runtimeNumber mirrors MikroORM's hydration of numeric columns mapped to a
// number property (e.g. a decimal link column with runtimeType number).
func runtimeNumber(propType string, v any) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	switch propType {
	case "number", "double", "float", "integer", "smallint", "decimal":
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
	}
	return v
}
