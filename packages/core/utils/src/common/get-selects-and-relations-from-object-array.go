package common

// GetSelectsAndRelationsFromObjectArray mirrors
// getSelectsAndRelationsFromObjectArray. Objects are given as ordered
// records through the ObjectLike interface to avoid an import cycle with
// dal.
type ObjectLike interface {
	Keys() []string
	Get(key string) (any, bool)
}

var keysThatAreNotRelations = map[string]bool{"metadata": true}

// SelectsAndRelationsOptions mirrors the options argument.
type SelectsAndRelationsOptions struct {
	ObjectFields   []string
	RequiredFields []string
}

func setKey(key, prefix string) string {
	if prefix != "" {
		return prefix + "." + key
	}
	return key
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func GetSelectsAndRelationsFromObjectArray(data []ObjectLike, opts SelectsAndRelationsOptions, prefix string) (selects, relations []string) {
	isObjectField := func(k string) bool {
		for _, f := range opts.ObjectFields {
			if f == k {
				return true
			}
		}
		return false
	}
	for _, d := range data {
		if d == nil {
			continue
		}
		for _, key := range d.Keys() {
			value, _ := d.Get(key)
			var nested []ObjectLike
			isNested := false
			switch v := value.(type) {
			case ObjectLike:
				if v != nil && !isObjectField(key) {
					nested, isNested = []ObjectLike{v}, true
				}
			case []any:
				isNested = true
				for _, x := range v {
					if o, ok := x.(ObjectLike); ok {
						nested = append(nested, o)
					}
				}
			}
			if !isNested {
				selects = append(selects, setKey(key, prefix))
				continue
			}
			s, r := GetSelectsAndRelationsFromObjectArray(nested, SelectsAndRelationsOptions{ObjectFields: opts.ObjectFields}, setKey(key, prefix))
			selects = append(selects, s...)
			if !keysThatAreNotRelations[key] {
				relations = append(relations, setKey(key, prefix))
				relations = append(relations, r...)
			}
		}
	}
	return dedupe(append(selects, opts.RequiredFields...)), dedupe(relations)
}
