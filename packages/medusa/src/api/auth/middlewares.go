// Package auth mirrors api/auth/middlewares.ts.
package auth

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	authutils "github.com/hitkey-io/medusago/packages/medusa/src/api/auth/utils"
)

func boolPtr(b bool) *bool { return &b }

// AuthRoutesMiddlewares mirrors authRoutesMiddlewares (the entries of the
// ported routes).
var AuthRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{
		Methods: []string{"POST"},
		Matcher: "/auth/session",
		Middlewares: []medusahttp.MiddlewareFunction{
			middlewares.Authenticate([]string{"*"}, []string{"bearer"}, medusahttp.AuthenticateOptions{AllowUnregistered: true, RequireMfa: boolPtr(false)}),
		},
	},
	{
		Methods: []string{"DELETE"},
		Matcher: "/auth/session",
		Middlewares: []medusahttp.MiddlewareFunction{
			middlewares.Authenticate([]string{"*"}, []string{"session"}, medusahttp.AuthenticateOptions{AllowUnregistered: true, RequireMfa: boolPtr(false)}),
		},
	},
	{
		Methods: []string{"POST"},
		Matcher: "/auth/token/refresh",
		Middlewares: []medusahttp.MiddlewareFunction{
			middlewares.Authenticate([]string{"*"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{AllowUnregistered: true, RequireMfa: boolPtr(false)}),
		},
	},
	{
		Methods:     []string{"GET", "POST"},
		Matcher:     "/auth/:actor_type/:auth_provider/callback",
		Middlewares: []medusahttp.MiddlewareFunction{authutils.ValidateScopeProviderAssociation("")},
	},
	{
		Methods:     []string{"POST"},
		Matcher:     "/auth/:actor_type/:auth_provider/register",
		Middlewares: []medusahttp.MiddlewareFunction{authutils.ValidateScopeProviderAssociation("")},
	},
	{
		Methods:     []string{"POST"},
		Matcher:     "/auth/:actor_type/:auth_provider",
		Middlewares: []medusahttp.MiddlewareFunction{authutils.ValidateScopeProviderAssociation("")},
	},
	{
		Methods:     []string{"GET"},
		Matcher:     "/auth/:actor_type/:auth_provider",
		Middlewares: []medusahttp.MiddlewareFunction{authutils.ValidateScopeProviderAssociation("")},
	},
	{
		Methods:     []string{"POST"},
		Matcher:     "/auth/:actor_type/:auth_provider/update",
		Middlewares: []medusahttp.MiddlewareFunction{authutils.ValidateScopeProviderAssociation("")},
	},
}
