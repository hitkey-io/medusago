package dal

import (
	"strconv"
	"sync"
)

// OnInitHook mirrors a MikroORM @OnInit hook: it runs when an entity is
// hydrated and can set computed values in its row.
type OnInitHook func(data Row)

var (
	hooksMu     sync.RWMutex
	onInitHooks = map[string][]OnInitHook{}
)

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
