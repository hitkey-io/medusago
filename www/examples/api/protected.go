package api

import (
	"strings"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// ptr returns a pointer to v. Route.Authenticate and Route.Cors are pointers
// so that "not set" (enabled) can be told apart from false.
func ptr[T any](v T) *T { return &v }

// region: opt-out
func adminHelloGET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return res.JSON(map[string]any{"message": "Hello"})
}

// AdminHelloRoute is created under /admin, but it opts out of the
// authentication requirement that routes with this prefix have by default.
var AdminHelloRoute = medusahttp.RouteFile{
	Path: "/admin/custom",
	Route: medusahttp.Route{
		GET:          adminHelloGET,
		Authenticate: ptr(false),
	},
}

// endregion: opt-out

// region: authenticate
// ProtectedMiddlewares protect custom API routes with the authenticate
// middleware.
var ProtectedMiddlewares = []medusahttp.MiddlewareRoute{
	{
		// Only authenticated admin users can access these routes.
		Matcher: "/custom/admin*",
		Middlewares: []medusahttp.MiddlewareFunction{
			middlewares.Authenticate(
				[]string{"user"},
				[]string{"session", "bearer", "api-key"},
				medusahttp.AuthenticateOptions{},
			),
		},
	},
	{
		// Only authenticated customers can access these routes.
		Matcher: "/custom/customer*",
		Middlewares: []medusahttp.MiddlewareFunction{
			middlewares.Authenticate(
				[]string{"customer"},
				[]string{"session", "bearer"},
				medusahttp.AuthenticateOptions{},
			),
		},
	},
}

// endregion: authenticate

// region: actor-types
// ActorTypeMiddlewares show how to require a custom actor type, or any of
// several actor types.
var ActorTypeMiddlewares = []medusahttp.MiddlewareRoute{
	{
		Matcher: "/manager*",
		Middlewares: []medusahttp.MiddlewareFunction{
			middlewares.Authenticate([]string{"manager"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{}),
		},
	},
	{
		Matcher: "/shared*",
		Middlewares: []medusahttp.MiddlewareFunction{
			middlewares.Authenticate([]string{"user", "customer"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{}),
		},
	},
}

// endregion: actor-types

// region: options
// OptionsMiddlewares show the options of the authenticate middleware.
var OptionsMiddlewares = []medusahttp.MiddlewareRoute{
	{
		Matcher: "/custom/optional*",
		Middlewares: []medusahttp.MiddlewareFunction{
			middlewares.Authenticate([]string{"customer"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{
				// Requests without credentials reach the route as guests.
				AllowUnauthenticated: true,
			}),
		},
	},
	{
		Matcher: "/custom/register*",
		Middlewares: []medusahttp.MiddlewareFunction{
			middlewares.Authenticate([]string{"manager"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{
				// Accept a registration token, for a user that has an
				// authentication identity but no actor yet.
				AllowUnregistered: true,
			}),
		},
	},
	{
		Matcher: "/custom/sensitive*",
		Middlewares: []medusahttp.MiddlewareFunction{
			middlewares.Authenticate([]string{"user"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{
				// Identities with MFA enabled must have completed a
				// challenge in the last ten minutes.
				RequireMfaMaxAgeSeconds: 600,
			}),
		},
	},
}

// endregion: options

// region: reject-invalid
// rejectInvalidAuth answers with a 401 error when a request presented
// credentials that the authenticate middleware couldn't verify.
func rejectInvalidAuth(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
	hasToken := strings.HasPrefix(strings.ToLower(req.Header.Get("Authorization")), "bearer ")
	_, hasSession := req.Session.Get("auth_context")

	if (hasToken || hasSession) && req.AuthContext == nil {
		return common.NewMedusaError(
			common.MedusaErrorTypes.UNAUTHORIZED,
			"Your session is invalid or has expired.",
		)
	}

	next(nil)
	return nil
}

// OptionalAuthMiddleware runs rejectInvalidAuth after the authenticate
// middleware, since it reads the authentication context it sets.
var OptionalAuthMiddleware = medusahttp.MiddlewareRoute{
	Matcher: "/custom/public*",
	Middlewares: []medusahttp.MiddlewareFunction{
		middlewares.Authenticate([]string{"customer"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{
			AllowUnauthenticated: true,
		}),
		rejectInvalidAuth,
	},
}

// endregion: reject-invalid

// region: actor-id
func whoamiGET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	// AuthContext is nil when nobody is authenticated.
	return res.JSON(map[string]any{
		"actor_id":   req.AuthString("actor_id"),
		"actor_type": req.AuthString("actor_type"),
	})
}

// endregion: actor-id

// region: customer
func customerProfileGET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filters := dal.NewRecord()
	filters.Set("id", req.AuthString("actor_id"))

	result, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{
		Entity:  "customer",
		Fields:  []string{"*"},
		Filters: filters,
	}, joiner.Options{
		// Return an error if the customer isn't found.
		ThrowIfKeyNotFound: true,
	})
	if err != nil {
		return err
	}

	return res.JSON(map[string]any{"customer": result.Data[0]})
}

// endregion: customer

// ProtectedRoutes are the route files of the protected routes chapter.
var ProtectedRoutes = []medusahttp.RouteFile{
	AdminHelloRoute,
	{Path: "/custom/admin/whoami", Route: medusahttp.Route{GET: whoamiGET}},
	{Path: "/custom/customer/profile", Route: medusahttp.Route{GET: customerProfileGET}},
}
