package dal

import "strings"

func pickList(v any) ([]any, bool) {
	switch t := v.(type) {
	case []any:
		return t, true
	case []*Record:
		out := make([]any, len(t))
		for i, r := range t {
			out[i] = r
		}
		return out, true
	}
	return nil, false
}

// PickValueFromObject mirrors pickValueFromObject: ok is false for
// undefined; arrays along the path are mapped and flattened.
func PickValueFromObject(path string, obj any) (any, bool) {
	segments := strings.Split(path, ".")
	result := obj
	for i, seg := range segments {
		r, isRec := result.(*Record)
		if !isRec || r == nil {
			return nil, false
		}
		v, ok := r.Get(seg)
		if !ok || v == nil {
			return nil, false
		}
		result = v
		if i == len(segments)-1 {
			return result, true
		}
		if list, isList := pickList(result); isList {
			sub := strings.Join(segments[i+1:], ".")
			out := []any{}
			for _, item := range list {
				val, ok := PickValueFromObject(sub, item)
				if l, isL := pickList(val); isL {
					out = append(out, l...)
				} else if ok {
					out = append(out, val)
				} else {
					out = append(out, nil)
				}
			}
			return out, true
		}
		if _, isObj := result.(*Record); !isObj {
			return result, true
		}
	}
	return result, true
}
