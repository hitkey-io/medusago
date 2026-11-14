package api

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
)

func passThrough(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
	next(nil)
	return nil
}

// Stand-ins for middleware functions that do something useful.
var globalMiddleware, routeMiddleware, a, b, c, d = passThrough, passThrough, passThrough, passThrough, passThrough, passThrough

// region: execution-order
// ExecutionOrderExample has a global middleware and a route middleware for
// the same path.
var ExecutionOrderExample = []medusahttp.MiddlewareRoute{
	{
		Matcher:     "/custom",
		Middlewares: []medusahttp.MiddlewareFunction{globalMiddleware},
	},
	{
		Matcher:     "/custom",
		Methods:     []string{"GET"},
		Middlewares: []medusahttp.MiddlewareFunction{routeMiddleware},
	},
}

// endregion: execution-order

// region: sorting
// SortingExample has middlewares with different matchers.
var SortingExample = []medusahttp.MiddlewareRoute{
	{Matcher: "/custom/:id", Middlewares: []medusahttp.MiddlewareFunction{a}},
	{Matcher: "/custom", Middlewares: []medusahttp.MiddlewareFunction{b}},
	{Matcher: "/custom*", Methods: []string{"GET"}, Middlewares: []medusahttp.MiddlewareFunction{c}},
	{Matcher: "/custom/:id", Methods: []string{"GET"}, Middlewares: []medusahttp.MiddlewareFunction{d}},
}

// endregion: sorting
