package http

import "strings"

// SortableRoute is what RoutesSorter orders: a matcher (string or regexp)
// and whether the entry has a method (route or method-bound middleware).
type SortableRoute interface {
	sortMatcher() (matcher string, isRegexp bool)
	sortHasMethod() bool
}

type routesBucket struct {
	routes   []SortableRoute
	children map[string]*routesBranch
	order    []string
}

type routesBranch struct {
	buckets map[string]*routesBucket
}

func newRoutesBranch() *routesBranch {
	b := &routesBranch{buckets: map[string]*routesBucket{}}
	for _, k := range []string{"global", "regex", "wildcard", "static", "params"} {
		b.buckets[k] = &routesBucket{}
	}
	return b
}

// DefaultRoutesOrder mirrors the default order of RoutesSorter.
var DefaultRoutesOrder = []string{"global", "wildcard", "regex", "static", "params"}

// SortRoutes mirrors RoutesSorter.sort.
func SortRoutes(routes []SortableRoute, orderBy []string) []SortableRoute {
	if orderBy == nil {
		orderBy = DefaultRoutesOrder
	}
	root := newRoutesBranch()
	for _, route := range routes {
		parent := root
		matcher, isRe := route.sortMatcher()
		if isRe {
			parent.buckets["regex"].routes = append(parent.buckets["regex"].routes, route)
			continue
		}
		var segments []string
		for _, s := range strings.Split(matcher, "/") {
			if s != "" {
				segments = append(segments, s)
			}
		}
		if len(segments) == 0 {
			bucket := "static"
			if !route.sortHasMethod() {
				bucket = "global"
			}
			parent.buckets[bucket].routes = append(parent.buckets[bucket].routes, route)
			continue
		}
		for i, segment := range segments {
			bucket := "static"
			switch {
			case !route.sortHasMethod():
				bucket = "global"
			case strings.HasPrefix(segment, "*"):
				bucket = "wildcard"
			case strings.HasPrefix(segment, ":"):
				bucket = "params"
			case strings.ContainsAny(segment, "(+*[])!"):
				bucket = "regex"
			}
			b := parent.buckets[bucket]
			if i+1 == len(segments) {
				b.routes = append(b.routes, route)
				break
			}
			if b.children == nil {
				b.children = map[string]*routesBranch{}
			}
			if _, ok := b.children[segment]; !ok {
				b.children[segment] = newRoutesBranch()
				b.order = append(b.order, segment)
			}
			parent = b.children[segment]
		}
	}
	return sortBranch([]*routesBranch{root}, orderBy)
}

// sortBranch mirrors #sortBranch: branches are visited in insertion order
// (JS object key order; segments are never integer-like here).
func sortBranch(branches []*routesBranch, orderBy []string) []SortableRoute {
	result := map[string][]SortableRoute{}
	for _, node := range branches {
		for _, k := range []string{"global", "wildcard", "regex", "static", "params"} {
			b := node.buckets[k]
			result[k] = append(result[k], b.routes...)
			if b.children != nil {
				children := make([]*routesBranch, 0, len(b.order))
				for _, seg := range jsKeyOrder(b.order) {
					children = append(children, b.children[seg])
				}
				result[k] = append(result[k], sortBranch(children, orderBy)...)
			}
		}
	}
	var out []SortableRoute
	for _, k := range orderBy {
		out = append(out, result[k]...)
	}
	return out
}

// jsKeyOrder orders keys like a JS object: integer-like keys first,
// ascending, then the others in insertion order.
func jsKeyOrder(keys []string) []string {
	var ints, others []string
	for _, k := range keys {
		if isArrayIndex(k) {
			ints = append(ints, k)
		} else {
			others = append(others, k)
		}
	}
	for i := 1; i < len(ints); i++ {
		for j := i; j > 0 && (len(ints[j]) < len(ints[j-1]) || (len(ints[j]) == len(ints[j-1]) && ints[j] < ints[j-1])); j-- {
			ints[j], ints[j-1] = ints[j-1], ints[j]
		}
	}
	return append(ints, others...)
}

func isArrayIndex(k string) bool {
	if k == "" || len(k) > 10 {
		return false
	}
	if k != "0" && k[0] == '0' {
		return false
	}
	for _, c := range k {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
