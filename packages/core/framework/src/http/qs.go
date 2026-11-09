package http

import (
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// ParseQS mirrors qs.parse(str, { allowPrototypes: true }) as used by the
// Express "extended" query parser. Objects are *dal.Record, arrays []any,
// leaves strings.
func ParseQS(str string) *dal.Record {
	return parseQS(str, qsOptions{depth: 5, arrayLimit: 20, parameterLimit: 1000})
}

// ParseQSBody mirrors the qs options of body-parser urlencoded (extended).
func ParseQSBody(str string) *dal.Record {
	return parseQS(str, qsOptions{depth: 32, arrayLimit: 100, parameterLimit: 1000})
}

type qsOptions struct {
	depth          int
	arrayLimit     int
	parameterLimit int
}

// sparseArray is a JS array that may have holes.
type sparseArray struct {
	items map[int]any
}

func (a *sparseArray) length() int {
	n := 0
	for k := range a.items {
		if k+1 > n {
			n = k + 1
		}
	}
	return n
}

func (a *sparseArray) push(v any) { a.items[a.length()] = v }

func (a *sparseArray) keys() []int {
	ks := make([]int, 0, len(a.items))
	for k := range a.items {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	return ks
}

func qsDecode(s string) string {
	s = strings.ReplaceAll(s, "+", " ")
	if d, err := url.PathUnescape(s); err == nil {
		return d
	}
	return s
}

func parseQS(str string, o qsOptions) *dal.Record {
	str = strings.TrimPrefix(str, "?")
	tempObj := dal.NewRecord()
	if str == "" {
		return tempObj
	}
	parts := strings.Split(str, "&")
	if len(parts) > o.parameterLimit {
		parts = parts[:o.parameterLimit]
	}
	for _, part := range parts {
		if part == "" {
			continue
		}
		bracketEqualsPos := strings.Index(part, "]=")
		pos := strings.Index(part, "=")
		if bracketEqualsPos != -1 {
			pos = bracketEqualsPos + 1
		}
		var key, val string
		if pos == -1 {
			key = qsDecode(part)
			val = ""
		} else {
			key = qsDecode(part[:pos])
			val = qsDecode(part[pos+1:])
		}
		if key == "" {
			continue
		}
		if existing, ok := tempObj.Get(key); ok {
			tempObj.Set(key, combine(existing, val))
		} else {
			tempObj.Set(key, val)
		}
	}

	var obj any = dal.NewRecord()
	for _, key := range tempObj.Keys() {
		v, _ := tempObj.Get(key)
		newObj := parseKeys(key, v, o)
		if newObj == nil {
			continue
		}
		obj = qsMerge(obj, newObj)
	}
	return compact(obj).(*dal.Record)
}

func combine(a, b any) any {
	var out []any
	if arr, ok := a.([]any); ok {
		out = append(out, arr...)
	} else {
		out = append(out, a)
	}
	if arr, ok := b.([]any); ok {
		out = append(out, arr...)
	} else {
		out = append(out, b)
	}
	return out
}

var (
	qsBrackets = regexp.MustCompile(`(\[[^[\]]*])`)
)

func parseKeys(key string, val any, o qsOptions) any {
	var keys []string
	segment := key
	if o.depth > 0 {
		if loc := qsBrackets.FindStringIndex(key); loc != nil {
			segment = key[:loc[0]]
		}
	}
	if segment != "" {
		keys = append(keys, segment)
	}
	matches := qsBrackets.FindAllStringIndex(key, -1)
	i := 0
	var pending []int
	for _, m := range matches {
		if !(o.depth > 0) {
			break
		}
		if i >= o.depth {
			pending = m
			break
		}
		i++
		keys = append(keys, key[m[0]:m[1]])
	}
	if pending != nil {
		keys = append(keys, "["+key[pending[0]:]+"]")
	}
	return parseObject(keys, val, o)
}

func parseObject(chain []string, val any, o qsOptions) any {
	leaf := val
	for i := len(chain) - 1; i >= 0; i-- {
		root := chain[i]
		var obj any
		if root == "[]" {
			arr := &sparseArray{items: map[int]any{}}
			if a, ok := leaf.([]any); ok {
				for _, x := range a {
					arr.push(x)
				}
			} else if a, ok := leaf.(*sparseArray); ok {
				arr = a
			} else {
				arr.push(leaf)
			}
			obj = arr
		} else {
			cleanRoot := root
			if strings.HasPrefix(root, "[") && strings.HasSuffix(root, "]") {
				cleanRoot = root[1 : len(root)-1]
			}
			index, err := strconv.Atoi(cleanRoot)
			if err == nil && root != cleanRoot && strconv.Itoa(index) == cleanRoot && index >= 0 && index <= o.arrayLimit {
				arr := &sparseArray{items: map[int]any{index: leaf}}
				obj = arr
			} else if cleanRoot != "__proto__" {
				r := dal.NewRecord()
				r.Set(cleanRoot, leaf)
				obj = r
			} else {
				obj = dal.NewRecord()
			}
		}
		leaf = obj
	}
	return leaf
}

func isQSObject(v any) bool {
	switch v.(type) {
	case *dal.Record, *sparseArray, []any:
		return true
	}
	return false
}

func toSparse(v any) (*sparseArray, bool) {
	switch t := v.(type) {
	case *sparseArray:
		return t, true
	case []any:
		a := &sparseArray{items: map[int]any{}}
		for i, x := range t {
			a.items[i] = x
		}
		return a, true
	}
	return nil, false
}

func arrayToObject(a *sparseArray) *dal.Record {
	r := dal.NewRecord()
	for _, k := range a.keys() {
		r.Set(strconv.Itoa(k), a.items[k])
	}
	return r
}

// qsMerge mirrors qs utils.merge (allowPrototypes: true).
func qsMerge(target, source any) any {
	if source == nil {
		return target
	}
	if !isQSObject(source) {
		if arr, ok := toSparse(target); ok {
			arr.push(source)
			return arr
		}
		if r, ok := target.(*dal.Record); ok {
			if s, ok := source.(string); ok {
				r.Set(s, true)
			}
			return r
		}
		return []any{target, source}
	}
	if !isQSObject(target) {
		out := &sparseArray{items: map[int]any{0: target}}
		if sa, ok := toSparse(source); ok {
			for _, k := range sa.keys() {
				out.push(sa.items[k])
			}
			return out
		}
		return []any{target, source}
	}
	ta, tIsArr := toSparse(target)
	sa, sIsArr := toSparse(source)
	if tIsArr && sIsArr {
		for _, i := range sa.keys() {
			item := sa.items[i]
			if cur, ok := ta.items[i]; ok {
				if isQSObject(cur) && isQSObject(item) {
					ta.items[i] = qsMerge(cur, item)
				} else {
					ta.push(item)
				}
			} else {
				ta.items[i] = item
			}
		}
		return ta
	}
	var mergeTarget *dal.Record
	if tIsArr {
		mergeTarget = arrayToObject(ta)
	} else {
		mergeTarget = target.(*dal.Record)
	}
	if sIsArr {
		for _, k := range sa.keys() {
			key := strconv.Itoa(k)
			if cur, ok := mergeTarget.Get(key); ok {
				mergeTarget.Set(key, qsMerge(cur, sa.items[k]))
			} else {
				mergeTarget.Set(key, sa.items[k])
			}
		}
		return mergeTarget
	}
	src := source.(*dal.Record)
	for _, key := range src.Keys() {
		v, _ := src.Get(key)
		if cur, ok := mergeTarget.Get(key); ok {
			mergeTarget.Set(key, qsMerge(cur, v))
		} else {
			mergeTarget.Set(key, v)
		}
	}
	return mergeTarget
}

// compact mirrors utils.compact: arrays lose their holes.
func compact(v any) any {
	switch t := v.(type) {
	case *sparseArray:
		out := make([]any, 0, len(t.items))
		for _, k := range t.keys() {
			out = append(out, compact(t.items[k]))
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = compact(x)
		}
		return out
	case *dal.Record:
		for _, k := range t.Keys() {
			x, _ := t.Get(k)
			t.Set(k, compact(x))
		}
		return t
	}
	return v
}
