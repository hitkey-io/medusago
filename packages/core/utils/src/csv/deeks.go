package csv

import (
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// deeksOptions mirrors deeks' DeeksOptions.
type deeksOptions struct {
	arrayIndexesAsKeys             bool
	expandNestedObjects            bool
	expandArrayObjects             bool
	ignoreEmptyArraysWhenExpanding bool
	escapeNestedDots               bool
	ignoreEmptyArrays              bool
}

// visitedSet mirrors the WeakSet of visited objects (records by identity).
type visitedSet map[*dal.Record]bool

func (v visitedSet) has(o any) bool {
	r, ok := o.(*dal.Record)
	return ok && v[r]
}

func (v visitedSet) add(o any) {
	if r, ok := o.(*dal.Record); ok {
		v[r] = true
	}
}

func unique(list []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range list {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func flatten(lists [][]string) []string {
	out := []string{}
	for _, l := range lists {
		out = append(out, l...)
	}
	return out
}

// isDocumentToRecurOn mirrors deeks' utils.isDocumentToRecurOn.
func isDocumentToRecurOn(v any) bool {
	r, ok := v.(*dal.Record)
	return ok && r != nil && r.Len() > 0
}

// deepKeysFromList mirrors deeks.deepKeysFromList.
func deepKeysFromList(list []any, opts deeksOptions, visited visitedSet) [][]string {
	if visited == nil {
		visited = visitedSet{}
	}
	out := make([][]string, len(list))
	for i, doc := range list {
		if isObject(doc) {
			if visited.has(doc) {
				out[i] = []string{}
				continue
			}
			out[i] = generateDeepKeysList("", doc, opts, visited)
		} else {
			out[i] = []string{}
		}
	}
	return out
}

type deeksFrame struct {
	obj      any
	keys     []string
	i        int
	basePath string
}

func generateDeepKeysList(heading string, data any, opts deeksOptions, visited visitedSet) []string {
	result := []string{}
	visited.add(data)
	stack := []*deeksFrame{{obj: data, keys: objectKeys(data), basePath: heading}}
	for len(stack) > 0 {
		frame := stack[len(stack)-1]
		if frame.i >= len(frame.keys) {
			stack = stack[:len(stack)-1]
			continue
		}
		currentKey := frame.keys[frame.i]
		frame.i++
		value := get(frame.obj, currentKey)
		keyName := buildKeyName(frame.basePath, escapeNestedDotsIfSpecified(currentKey, opts))
		arr, isArr := asArray(value)
		switch {
		case (opts.expandNestedObjects && isDocumentToRecurOn(value)) || (opts.arrayIndexesAsKeys && isArr && len(arr) > 0):
			if opts.arrayIndexesAsKeys && isArr {
				for idx := len(arr) - 1; idx >= 0; idx-- {
					elem := arr[idx]
					elemPath := buildKeyName(keyName, itoa(idx))
					if r, ok := elem.(*dal.Record); ok && r != nil {
						if !visited.has(r) {
							visited.add(r)
							stack = append(stack, &deeksFrame{obj: r, keys: r.Keys(), basePath: elemPath})
						}
					} else {
						result = append(result, elemPath)
					}
				}
			} else if !visited.has(value) {
				visited.add(value)
				stack = append(stack, &deeksFrame{obj: value, keys: objectKeys(value), basePath: keyName})
			}
		case opts.expandArrayObjects && isArr:
			result = append(result, processArrayKeys(arr, keyName, opts, visited)...)
		case opts.ignoreEmptyArrays && isArr && len(arr) == 0:
		default:
			result = append(result, keyName)
		}
	}
	return unique(result)
}

func processArrayKeys(subArray []any, currentKeyPath string, opts deeksOptions, visited visitedSet) []string {
	subArrayKeys := deepKeysFromList(subArray, opts, visited)
	if len(subArray) == 0 {
		if opts.ignoreEmptyArraysWhenExpanding {
			return []string{}
		}
		return []string{currentKeyPath}
	}
	if len(flatten(subArrayKeys)) == 0 {
		return []string{currentKeyPath}
	}
	mapped := make([][]string, len(subArrayKeys))
	for i, schemaKeys := range subArrayKeys {
		if len(schemaKeys) == 0 {
			mapped[i] = []string{currentKeyPath}
			continue
		}
		keys := make([]string, len(schemaKeys))
		for j, subKey := range schemaKeys {
			keys[j] = buildKeyName(currentKeyPath, escapeNestedDotsIfSpecified(subKey, opts))
		}
		mapped[i] = keys
	}
	return unique(flatten(mapped))
}

func escapeNestedDotsIfSpecified(key string, opts deeksOptions) string {
	if opts.escapeNestedDots {
		return strings.ReplaceAll(key, ".", "\\.")
	}
	return key
}

func buildKeyName(upper, current string) string {
	if upper != "" {
		return upper + "." + current
	}
	return current
}

// DeeksOptions mirrors deeks' DeeksOptions.
type DeeksOptions struct {
	ArrayIndexesAsKeys             bool
	ExpandNestedObjects            bool
	ExpandArrayObjects             bool
	IgnoreEmptyArraysWhenExpanding bool
	EscapeNestedDots               bool
	IgnoreEmptyArrays              bool
}

// DeepKeysFromList mirrors deeks.deepKeysFromList.
func DeepKeysFromList(list []any, o DeeksOptions) [][]string {
	return deepKeysFromList(list, deeksOptions{
		arrayIndexesAsKeys: o.ArrayIndexesAsKeys, expandNestedObjects: o.ExpandNestedObjects, expandArrayObjects: o.ExpandArrayObjects,
		ignoreEmptyArraysWhenExpanding: o.IgnoreEmptyArraysWhenExpanding, escapeNestedDots: o.EscapeNestedDots, ignoreEmptyArrays: o.IgnoreEmptyArrays,
	}, nil)
}
