package joiner

import "github.com/hitkey-io/medusago/packages/core/utils/src/dal"

// resolveFieldAliasEntry mirrors resolveFieldAliasEntry.
func resolveFieldAliasEntry(entries []FieldAliasEntry, currentPathEntity string) *FieldAliasEntry {
	if len(entries) == 0 {
		return nil
	}
	if len(entries) == 1 {
		e := entries[0]
		return &e
	}
	if currentPathEntity == "" {
		e := entries[0]
		return &e
	}
	for _, e := range entries {
		if e.Entity == currentPathEntity {
			found := e
			return &found
		}
	}
	e := entries[0]
	return &e
}

// getNestedItems mirrors getNestedItems.
func getNestedItems(items []any, property string) []any {
	var result []any
	for _, item := range items {
		rec, ok := item.(*dal.Record)
		if !ok || rec == nil {
			continue
		}
		v, _ := rec.Get(property)
		switch t := v.(type) {
		case nil:
		case []any:
			for _, x := range t {
				if x != nil {
					result = append(result, x)
				}
			}
		default:
			result = append(result, t)
		}
	}
	return result
}
