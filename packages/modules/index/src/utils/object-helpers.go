package utils

import (
	"sort"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// The object helpers of @medusajs/utils (common) the index module relies
// on, over ordered records (dal imports common, so they live here).

// isObject mirrors isObject: a plain object (not an array).
func isObject(v any) bool {
	_, ok := v.(*dal.Record)
	return ok
}

func get(r *dal.Record, key string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(key)
	return v
}

// spread returns { ...a, ...b }.
func spread(a, b *dal.Record) *dal.Record {
	out := dal.NewRecord()
	for _, r := range []*dal.Record{a, b} {
		if r == nil {
			continue
		}
		for _, k := range r.Keys() {
			v, _ := r.Get(k)
			out.Set(k, v)
		}
	}
	return out
}

// UnflattenObjectKeys mirrors unflattenObjectKeys.
func UnflattenObjectKeys(flattened *dal.Record) *dal.Record {
	result := dal.NewRecord()
	if flattened == nil {
		return result
	}
	for _, key := range flattened.Keys() {
		if strings.Contains(key, ".") {
			continue
		}
		v, _ := flattened.Get(key)
		if cur, ok := get(result, key).(*dal.Record); ok {
			vr, _ := v.(*dal.Record)
			result.Set(key, spread(cur, vr))
		} else {
			result.Set(key, v)
		}
	}
	for _, key := range flattened.Keys() {
		if !strings.Contains(key, ".") {
			continue
		}
		value, _ := flattened.Get(key)
		keys := strings.Split(key, ".")
		current := result
		for i, part := range keys {
			if i == len(keys)-1 {
				existing, has := current.Get(part)
				if vr, ok := value.(*dal.Record); ok && has && existing != nil && existing != false && existing != "" {
					er, _ := existing.(*dal.Record)
					current.Set(part, spread(er, vr))
				} else {
					current.Set(part, value)
				}
				continue
			}
			next, ok := get(current, part).(*dal.Record)
			if !ok {
				if v, has := current.Get(part); has && v != nil {
					// a primitive cannot hold properties: keep descending into
					// a detached object (the assignment is lost)
					next = dal.NewRecord()
				} else {
					next = dal.NewRecord()
					current.Set(part, next)
				}
			}
			current = next
		}
	}
	return result
}

// DeepMerge mirrors deepMerge(target, source).
func DeepMerge(target, source any) any {
	copy := dal.NewRecord()
	if t, ok := target.(*dal.Record); ok {
		copy = spread(t, nil)
	}
	return deepMergeRecursive(copy, source)
}

func deepMergeRecursive(a, b any) any {
	ar, ok := a.(*dal.Record)
	if !ok {
		return b
	}
	br, ok := b.(*dal.Record)
	if !ok {
		return a
	}
	for _, key := range br.Keys() {
		bv, _ := br.Get(key)
		if isObject(bv) {
			if v, has := ar.Get(key); !has || v == nil {
				ar.Set(key, dal.NewRecord())
			}
			av, _ := ar.Get(key)
			ar.Set(key, DeepMerge(av, bv))
		} else {
			ar.Set(key, bv)
		}
	}
	return ar
}

// ObjectFromStringPath mirrors objectFromStringPath (the collection is
// sorted in place).
func ObjectFromStringPath(collection []string) *dal.Record {
	sort.Strings(collection)
	output := dal.NewRecord()
	for _, relation := range collection {
		if relation == "" {
			continue
		}
		nested := strings.Split(relation, ".")
		if common.HasUnsafePathSegment(nested) {
			continue
		}
		if len(nested) > 1 {
			parent := output
			for len(nested) > 1 {
				n := nested[0]
				nested = nested[1:]
				next, ok := get(parent, n).(*dal.Record)
				if !ok {
					next = dal.NewRecord()
				}
				parent.Set(n, next)
				parent = next
			}
			parent.Set(nested[0], true)
			continue
		}
		if v, has := output.Get(relation); !has || v == nil {
			output.Set(relation, true)
		}
	}
	return output
}

// NormalizeFieldsSelection mirrors normalizeFieldsSelection.
func NormalizeFieldsSelection(fields []string) *dal.Record {
	normalized := make([]string, len(fields))
	for i, f := range fields {
		normalized[i] = strings.ReplaceAll(f, ".*", "")
	}
	return ObjectFromStringPath(normalized)
}

// FlattenObjectKeys mirrors flattenObjectKeys.
func FlattenObjectKeys(input *dal.Record) *dal.Record {
	result := dal.NewRecord()
	var flatten func(obj *dal.Record, path string, hasPath bool)
	flatten = func(obj *dal.Record, path string, hasPath bool) {
		for _, key := range obj.Keys() {
			v, _ := obj.Get(key)
			if _, isOp := OperatorMap[key]; isOp {
				if hasPath {
					result.Set(path, obj)
				} else {
					result.Set(key, obj)
				}
				continue
			}
			newPath := key
			if hasPath {
				newPath = path + "." + key
			}
			switch t := v.(type) {
			case nil:
				result.Set(newPath, nil)
			case []any:
				result.Set(newPath, t)
			case *dal.Record:
				flatten(t, newPath, true)
			default:
				result.Set(newPath, v)
			}
		}
	}
	if input != nil {
		flatten(input, "", false)
	}
	return result
}
