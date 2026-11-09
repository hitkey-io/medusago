package users

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// AdminUserRoutesMiddlewares mirrors adminUserRoutesMiddlewares (ported
// routes only; authorize is RBAC and not ported).
var AdminUserRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/users", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetUsersParams, ListTransformQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/users/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetUserParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/users/me", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetUserParams, RetrieveTransformQueryConfig)}},
}
