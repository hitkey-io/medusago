package http

import (
	"regexp"
	"strings"
	"sync"
)

// FindableRoute is what RoutesFinder searches.
type FindableRoute interface {
	findMatcher() (matcher string, re *regexp.Regexp)
	// findMethods returns the methods the entry applies to.
	findMethods() []string
}

type finderEntry struct {
	route FindableRoute
	re    *regexp.Regexp
}

// RoutesFinder mirrors RoutesFinder: the first added entry whose methods
// include the method and whose matcher (path-to-regexp 8) matches the path.
type RoutesFinder[T FindableRoute] struct {
	mu      sync.Mutex
	routes  []finderEntry
	matches map[string]*T
}

// NewRoutesFinder creates a finder over routes.
func NewRoutesFinder[T FindableRoute](routes []T) *RoutesFinder[T] {
	f := &RoutesFinder[T]{matches: map[string]*T{}}
	for _, r := range routes {
		f.Add(r)
	}
	return f
}

var (
	splatSlash = regexp.MustCompile(`/\*`)
	splatWord  = regexp.MustCompile(`(\w)\*`)
)

// Add mirrors RoutesFinder.add.
func (f *RoutesFinder[T]) Add(route T) {
	matcher, re := route.findMatcher()
	if re == nil {
		p := splatSlash.ReplaceAllString(matcher, "{*splat}")
		p = splatWord.ReplaceAllString(p, "$1{*splat}")
		re = compilePathToRegexp8(p)
	}
	f.mu.Lock()
	f.routes = append(f.routes, finderEntry{route: route, re: re})
	f.matches = map[string]*T{}
	f.mu.Unlock()
}

// Find mirrors RoutesFinder.find.
func (f *RoutesFinder[T]) Find(url, method string) (T, bool) {
	var zero T
	key := method + ":" + url
	f.mu.Lock()
	defer f.mu.Unlock()
	if m, ok := f.matches[key]; ok {
		if m == nil {
			return zero, false
		}
		return *m, true
	}
	for _, e := range f.routes {
		methods := e.route.findMethods()
		has := false
		for _, m := range methods {
			if m == method {
				has = true
				break
			}
		}
		if has && e.re.MatchString(url) {
			r := e.route.(T)
			f.matches[key] = &r
			return r, true
		}
	}
	f.matches[key] = nil
	return zero, false
}

// compilePathToRegexp8 mirrors pathToRegexp(path) of path-to-regexp 8
// (sensitive false, end true, trailing true) for the syntax Medusa uses:
// :param, *wildcard and {optional} groups.
func compilePathToRegexp8(path string) *regexp.Regexp {
	var b strings.Builder
	i := 0
	name := func() string {
		j := i
		for j < len(path) && (path[j] == '_' || path[j] == '$' || (path[j] >= '0' && path[j] <= '9') || (path[j] >= 'a' && path[j] <= 'z') || (path[j] >= 'A' && path[j] <= 'Z')) {
			j++
		}
		n := path[i:j]
		i = j
		return n
	}
	for i < len(path) {
		c := path[i]
		switch c {
		case '{':
			b.WriteString("(?:")
			i++
		case '}':
			b.WriteString(")?")
			i++
		case ':':
			i++
			name()
			b.WriteString(`([^/]+)`)
		case '*':
			i++
			name()
			b.WriteString(`([\s\S]+)`)
		case '\\':
			if i+1 < len(path) {
				b.WriteString(regexp.QuoteMeta(string(path[i+1])))
			}
			i += 2
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
			i++
		}
	}
	return regexp.MustCompile("(?i)^(?:" + b.String() + `)(?:/$)?$`)
}
