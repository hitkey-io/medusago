package api

import (
	"strings"
	"testing"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
)

// TestMiddlewareOrder checks what the middlewares chapter says about the
// order middlewares and handlers run in.
func TestMiddlewareOrder(t *testing.T) {
	var calls []string
	record := func(name string) medusahttp.MiddlewareFunction {
		return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
			calls = append(calls, name)
			next(nil)
			return nil
		}
	}
	mw := func(matcher string, methods []string, name string) medusahttp.MiddlewareRoute {
		return medusahttp.MiddlewareRoute{Matcher: matcher, Methods: methods, Middlewares: []medusahttp.MiddlewareFunction{record(name)}}
	}
	routes := []medusahttp.RouteFile{{Path: "/custom/:id", Route: medusahttp.Route{GET: func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
		calls = append(calls, "handler")
		return res.JSON(map[string]any{})
	}}}}
	srv := newServer(t, routes, []medusahttp.MiddlewareRoute{
		mw("/custom/:id", []string{"GET"}, "route /custom/:id"),
		mw("/custom/:id", nil, "global /custom/:id"),
		mw("/custom*", []string{"GET"}, "route /custom*"),
		mw("/custom*", nil, "global /custom*"),
	})

	do(t, "GET", srv.URL+"/custom/1", "", nil)

	got := strings.Join(calls, ", ")
	want := "global /custom*, global /custom/:id, route /custom*, route /custom/:id, handler"
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}

	// a middleware matcher accepts a request URL with a trailing slash
	calls = nil
	do(t, "GET", srv.URL+"/custom/1/", "", nil)
	if len(calls) != 5 {
		t.Fatalf("unexpected calls for a trailing slash: %v", calls)
	}
}

func TestRegexpMatcher(t *testing.T) {
	var calls int
	count := func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		calls++
		next(nil)
		return nil
	}
	handler := func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
		return res.JSON(map[string]any{})
	}
	routes := []medusahttp.RouteFile{
		{Path: "/custom", Route: medusahttp.Route{GET: handler}},
		{Path: "/other/custom", Route: medusahttp.Route{GET: handler}},
	}
	srv := newServer(t, routes, []medusahttp.MiddlewareRoute{{
		MatcherRe:   RegexpMiddleware.MatcherRe,
		Middlewares: []medusahttp.MiddlewareFunction{count},
	}})
	do(t, "GET", srv.URL+"/custom", "", nil)
	do(t, "GET", srv.URL+"/other/custom", "", nil)
	if calls != 1 {
		t.Fatalf("the anchored expression must only match /custom: %d", calls)
	}
}

func TestSortingExample(t *testing.T) {
	var calls []string
	names := []string{"a", "b", "c", "d"}
	mws := make([]medusahttp.MiddlewareRoute, len(SortingExample))
	for i, m := range SortingExample {
		name := names[i]
		m.Middlewares = []medusahttp.MiddlewareFunction{func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
			calls = append(calls, name)
			next(nil)
			return nil
		}}
		mws[i] = m
	}
	routes := []medusahttp.RouteFile{{Path: "/custom/:id", Route: medusahttp.Route{GET: func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
		return res.JSON(map[string]any{})
	}}}}
	srv := newServer(t, routes, mws)
	do(t, "GET", srv.URL+"/custom/1", "", nil)
	if got := strings.Join(calls, ","); got != "b,a,c,d" {
		t.Fatalf("got %s, want b,a,c,d", got)
	}
}
