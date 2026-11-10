package utils

import (
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func arrayOf(v any) ([]any, bool) {
	switch t := v.(type) {
	case []any:
		return t, true
	case []*dal.Record:
		out := make([]any, len(t))
		for i, r := range t {
			out[i] = r
		}
		return out, true
	}
	return nil, false
}

// FilterObjectByKeys mirrors filterObjectByKeys.
func FilterObjectByKeys(obj *dal.Record, paths []string) *dal.Record {
	result := dal.NewRecord()
	if obj == nil {
		return result
	}
	for _, path := range paths {
		parts := strings.Split(path, ".")
		if len(parts) == 1 {
			if v, ok := obj.Get(parts[0]); ok {
				result.Set(parts[0], v)
			}
			continue
		}
		current := result
		var source any = obj
		for i, part := range parts {
			isLast := i == len(parts)-1
			src, _ := source.(*dal.Record)
			if current == nil || src == nil {
				break
			}
			sv, defined := src.Get(part)
			if existing, _ := current.Get(part); existing == nil {
				if arr, ok := arrayOf(sv); ok {
					items := make([]any, len(arr))
					for j := range arr {
						items[j] = dal.NewRecord()
					}
					current.Set(part, items)
				} else if defined && sv == nil {
					current.Set(part, nil)
				} else if defined {
					current.Set(part, dal.NewRecord())
				}
			}
			if arr, ok := arrayOf(sv); ok {
				arrayPath := strings.Join(parts[:i+1], ".")
				var rel []string
				for _, p := range paths {
					if strings.HasPrefix(p, arrayPath+".") {
						rel = append(rel, p[len(arrayPath)+1:])
					}
				}
				cur, _ := current.Get(part)
				curArr, _ := cur.([]any)
				items := make([]any, len(arr))
				for j, item := range arr {
					var ex *dal.Record
					if j < len(curArr) {
						ex, _ = curArr[j].(*dal.Record)
					}
					if ex == nil {
						ex = dal.NewRecord()
					}
					for _, sub := range rel {
						keys := strings.Split(sub, ".")
						var val any = item
						found := true
						for _, k := range keys {
							r, ok := val.(*dal.Record)
							if !ok {
								found = false
								break
							}
							val, ok = r.Get(k)
							if !ok {
								found = false
								break
							}
						}
						if !found {
							continue
						}
						t := ex
						for _, k := range keys[:len(keys)-1] {
							n, _ := t.Get(k)
							nr, ok := n.(*dal.Record)
							if !ok || nr == nil {
								nr = dal.NewRecord()
								t.Set(k, nr)
							}
							t = nr
						}
						t.Set(keys[len(keys)-1], val)
					}
					items[j] = ex
				}
				current.Set(part, items)
				break
			}
			if isLast {
				if defined {
					current.Set(part, sv)
				}
			} else {
				next, _ := current.Get(part)
				current, _ = next.(*dal.Record)
				source = sv
			}
		}
	}
	return result
}
