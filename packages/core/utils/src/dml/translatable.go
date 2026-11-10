package dml

import "sync"

// TranslatableEntityEntry mirrors TranslatableEntityEntry: an entity name
// (PascalCase) and its translatable text properties.
type TranslatableEntityEntry struct {
	Entity string
	Fields []string
}

var (
	translatableMu       sync.Mutex
	translatableEntities []TranslatableEntityEntry
)

// registerTranslatableFields mirrors DmlEntity#registerTranslatableFields:
// every parsed entity with .translatable() text properties is registered,
// in parse order.
func registerTranslatableFields(e *Entity) {
	var fields []string
	for _, p := range e.Properties {
		if p.Field == nil || p.Field.DataType.Options == nil || !p.Field.DataType.Options.Translatable {
			continue
		}
		fields = append(fields, p.Name)
	}
	if len(fields) == 0 {
		return
	}
	translatableMu.Lock()
	defer translatableMu.Unlock()
	for i, entry := range translatableEntities {
		if entry.Entity != e.Name {
			continue
		}
		for _, f := range fields {
			if !containsString(entry.Fields, f) {
				translatableEntities[i].Fields = append(translatableEntities[i].Fields, f)
			}
		}
		return
	}
	translatableEntities = append(translatableEntities, TranslatableEntityEntry{Entity: e.Name, Fields: fields})
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// GetTranslatableEntities mirrors DmlEntity.getTranslatableEntities. The
// entries are in parse (Go package initialization) order; upstream orders
// them by module import order, which callers needing that order restore.
func GetTranslatableEntities() []TranslatableEntityEntry {
	translatableMu.Lock()
	defer translatableMu.Unlock()
	out := make([]TranslatableEntityEntry, len(translatableEntities))
	for i, e := range translatableEntities {
		out[i] = TranslatableEntityEntry{Entity: e.Entity, Fields: append([]string(nil), e.Fields...)}
	}
	return out
}
