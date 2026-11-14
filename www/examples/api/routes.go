package api

import (
	cli "github.com/hitkey-io/medusago/packages/cli/medusa-cli/src"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
)

// region: routes
// Routes lists every route file of the project.
var Routes = concat(
	[]medusahttp.RouteFile{HelloWorldRoute, SumRoute, CreatePostRoute, WebhookRoute, StoreCustomRoute},
	ParameterRoutes,
	ResponseRoutes,
	ErrorRoutes,
	ProtectedRoutes,
)

// Middlewares lists every middleware configuration of the project.
var Middlewares = concat(
	MiddlewaresExample,
	SumMiddlewares,
	CreatePostMiddlewares,
	WebhookMiddlewares,
	ProtectedMiddlewares,
	StoreCustomMiddlewares,
	[]medusahttp.MiddlewareRoute{CorsMiddleware},
)

// concat joins slices.
func concat[T any](lists ...[]T) []T {
	var out []T
	for _, l := range lists {
		out = append(out, l...)
	}
	return out
}

// endregion: routes

// region: options
// Options are the options main.go passes to the CLI.
func Options() cli.Options {
	return cli.Options{
		Config:      medusaConfig,
		Routes:      Routes,
		Middlewares: Middlewares,
	}
}

// endregion: options
