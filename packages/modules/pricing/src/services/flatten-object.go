package services

import (
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// keyValue is one flattened entry (in insertion order).
type keyValue struct {
	Key   string
	Value any
}

type flatResult struct {
	keys   []string
	values map[string]any
}

func (r *flatResult) set(path []string, v any) {
	k := strings.Join(path, ".")
	if _, ok := r.values[k]; !ok {
		r.keys = append(r.keys, k)
	}
	r.values[k] = v
}

func isObj(v any) bool {
	_, ok := v.(*dal.Record)
	return ok
}

func isArr(v any) bool {
	_, ok := v.([]any)
	return ok
}

func jsTypeOf(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case float64, int, int64:
		return "number"
	case bool:
		return "boolean"
	}
	return "object"
}

func shouldPreserveArray(value []any, path []string) bool {
	if len(value) == 0 {
		return false
	}
	for _, it := range value {
		if isObj(it) || isArr(it) {
			return true
		}
	}
	if len(path) > 1 {
		return true
	}
	if len(value) > 1 {
		first := jsTypeOf(value[0])
		all := true
		for _, it := range value {
			if jsTypeOf(it) != first {
				all = false
			}
		}
		if all {
			return true
		}
	}
	return false
}

func normalizeArrayValue(v any, path []string) any {
	arr, ok := v.([]any)
	if !ok {
		return v
	}
	if len(arr) == 1 {
		if inner, ok := arr[0].([]any); ok && len(inner) == 0 {
			return []any{}
		}
	}
	if shouldPreserveArray(arr, path) {
		return arr
	}
	if len(arr) == 1 && !isObj(arr[0]) && !isArr(arr[0]) {
		return arr[0]
	}
	return arr
}

func copyPath(p []string, k string) []string {
	out := append([]string{}, p...)
	return append(out, k)
}

// flattenObjectToKeyValuePairs mirrors flattenObjectToKeyValuePairs.
func flattenObjectToKeyValuePairs(obj *dal.Record) []keyValue {
	r := &flatResult{values: map[string]any{}}
	var processPath func(v any, path []string)
	var extractFromArray func(arr []any, base []string)
	var extractNested func(objs []any, base []string)

	processPath = func(value any, path []string) {
		switch t := value.(type) {
		case nil:
			return
		case *dal.Record:
			for _, k := range t.Keys() {
				pv, _ := t.Get(k)
				np := copyPath(path, k)
				switch x := pv.(type) {
				case nil:
				case []any:
					if len(x) == 0 {
						r.set(np, []any{})
					} else {
						processPath(x, np)
					}
				case *dal.Record:
					processPath(x, np)
				default:
					r.set(np, x)
				}
			}
		case []any:
			if len(t) == 0 {
				if len(path) > 0 {
					r.set(path, []any{})
				}
				return
			}
			hasObj, hasArr := false, false
			for _, it := range t {
				if isObj(it) {
					hasObj = true
				}
				if isArr(it) {
					hasArr = true
				}
			}
			switch {
			case hasObj:
				extractFromArray(t, path)
			case hasArr:
				var all, objs []any
				var flat func(a []any)
				flat = func(a []any) {
					for _, it := range a {
						switch x := it.(type) {
						case []any:
							flat(x)
						case *dal.Record:
							objs = append(objs, x)
						case nil:
						default:
							all = append(all, x)
						}
					}
				}
				flat(t)
				if len(objs) > 0 {
					extractFromArray(objs, path)
				} else if len(all) > 0 {
					r.set(path, normalizeArrayValue(all, path))
				}
			default:
				var cleaned []any
				for _, it := range t {
					if it != nil {
						cleaned = append(cleaned, it)
					}
				}
				if len(cleaned) > 0 {
					r.set(path, normalizeArrayValue(cleaned, path))
				}
			}
		default:
			if len(path) > 0 {
				if s, ok := t.(string); ok && s == "" {
					return
				}
				if b, ok := t.(bool); ok && !b {
					return
				}
				if f, ok := t.(float64); ok && f == 0 {
					return
				}
				r.set(path, t)
			}
		}
	}

	collectKeys := func(items []any) []string {
		var keys []string
		seen := map[string]bool{}
		for _, it := range items {
			if rec, ok := it.(*dal.Record); ok {
				for _, k := range rec.Keys() {
					if !seen[k] {
						seen[k] = true
						keys = append(keys, k)
					}
				}
			}
		}
		return keys
	}
	valuesOf := func(items []any, key string) []any {
		var vals []any
		for _, it := range items {
			if rec, ok := it.(*dal.Record); ok {
				if v, ok := rec.Get(key); ok && v != nil {
					vals = append(vals, v)
				}
			}
		}
		return vals
	}
	allObjects := func(vals []any) bool {
		for _, v := range vals {
			if !isObj(v) {
				return false
			}
		}
		return true
	}
	anyArray := func(vals []any) bool {
		for _, v := range vals {
			if isArr(v) {
				return true
			}
		}
		return false
	}
	allEmptyArrays := func(vals []any) bool {
		for _, v := range vals {
			a, ok := v.([]any)
			if !ok || len(a) > 0 {
				return false
			}
		}
		return true
	}

	extractFromArray = func(arr []any, base []string) {
		if len(arr) == 0 {
			return
		}
		for _, key := range collectKeys(arr) {
			vp := copyPath(base, key)
			vals := valuesOf(arr, key)
			if len(vals) == 0 {
				continue
			}
			switch {
			case allObjects(vals):
				extractNested(vals, vp)
			case anyArray(vals):
				if allEmptyArrays(vals) {
					r.set(vp, []any{})
					continue
				}
				var flat []any
				for _, av := range vals {
					if a, ok := av.([]any); ok {
						hasObj := false
						for _, x := range a {
							if isObj(x) {
								hasObj = true
							}
						}
						if hasObj {
							extractFromArray(a, vp)
						} else {
							flat = append(flat, a...)
						}
					} else {
						flat = append(flat, av)
					}
				}
				hasObj := false
				for _, x := range flat {
					if isObj(x) {
						hasObj = true
					}
				}
				if len(flat) > 0 && !hasObj {
					r.set(vp, normalizeArrayValue(flat, vp))
				}
			default:
				r.set(vp, normalizeArrayValue(vals, vp))
			}
		}
	}

	extractNested = func(objs []any, base []string) {
		if len(objs) == 0 {
			return
		}
		for _, key := range collectKeys(objs) {
			np := copyPath(base, key)
			vals := valuesOf(objs, key)
			if len(vals) == 0 {
				continue
			}
			switch {
			case allObjects(vals):
				extractNested(vals, np)
			case anyArray(vals):
				if allEmptyArrays(vals) {
					r.set(np, []any{})
					continue
				}
				var items []any
				for _, av := range vals {
					if a, ok := av.([]any); ok {
						items = append(items, a...)
					} else {
						items = append(items, av)
					}
				}
				hasObj := false
				for _, x := range items {
					if isObj(x) {
						hasObj = true
					}
				}
				if hasObj {
					extractFromArray(items, np)
				} else {
					r.set(np, normalizeArrayValue(items, np))
				}
			default:
				r.set(np, normalizeArrayValue(vals, np))
			}
		}
	}

	processPath(obj, nil)
	out := make([]keyValue, len(r.keys))
	for i, k := range r.keys {
		out[i] = keyValue{Key: k, Value: r.values[k]}
	}
	return out
}
