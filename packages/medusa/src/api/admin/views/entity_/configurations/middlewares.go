package configurations

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/views/entity_/columns"
)

func mws(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// ViewConfigurationRoutesMiddlewares mirrors viewConfigurationRoutesMiddlewares
// (ensureViewEntityReadable and requireSuperAdmin are no-ops without RBAC).
var ViewConfigurationRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET", "POST", "DELETE"}, Matcher: "/admin/views/*/configurations*", Middlewares: mws(EnsureViewConfigurationsEnabled)},
	{Methods: []string{"GET"}, Matcher: "/admin/views/:entity/configurations", Middlewares: mws(
		EnsureViewEntityReadable,
		httputils.ValidateAndTransformQuery(AdminGetViewConfigurationsParams, RetrieveViewConfigurationList),
	)},
	{Methods: []string{"POST"}, Matcher: "/admin/views/:entity/configurations", Middlewares: mws(
		EnsureViewEntityReadable,
		httputils.ValidateAndTransformQuery(AdminGetViewConfigurationParams, RetrieveViewConfiguration),
		httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateViewConfiguration)),
		RequireSuperAdmin,
	)},
	{Methods: []string{"GET"}, Matcher: "/admin/views/:entity/configurations/:id", Middlewares: mws(
		EnsureViewEntityReadable,
		httputils.ValidateAndTransformQuery(AdminGetViewConfigurationParams, RetrieveViewConfiguration),
	)},
	{Methods: []string{"POST"}, Matcher: "/admin/views/:entity/configurations/:id", Middlewares: mws(
		EnsureViewEntityReadable,
		httputils.ValidateAndTransformQuery(AdminGetViewConfigurationParams, RetrieveViewConfiguration),
		httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateViewConfiguration)),
		RequireSuperAdmin,
	)},
	{Methods: []string{"GET"}, Matcher: "/admin/views/:entity/configurations/active", Middlewares: mws(
		EnsureViewEntityReadable,
		httputils.ValidateAndTransformQuery(AdminGetActiveViewConfigurationParams, RetrieveViewConfiguration),
	)},
	{Methods: []string{"POST"}, Matcher: "/admin/views/:entity/configurations/active", Middlewares: mws(
		EnsureViewEntityReadable,
		httputils.ValidateAndTransformBody(httputils.Schema(AdminSetActiveViewConfiguration)),
	)},
	{Methods: []string{"DELETE"}, Matcher: "/admin/views/:entity/configurations/:id", Middlewares: mws(
		EnsureViewEntityReadable,
		RequireSuperAdmin,
	)},
}

// EntitiesRoutesMiddlewares mirrors entitiesRoutesMiddlewares.
var EntitiesRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/views/entities", Middlewares: mws(EnsureViewConfigurationsEnabled)},
}

// ColumnRoutesMiddlewares mirrors columnRoutesMiddlewares.
var ColumnRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET", "POST"}, Matcher: "/admin/views/*/columns", Middlewares: mws(EnsureViewConfigurationsEnabled)},
	{Methods: []string{"GET"}, Matcher: "/admin/views/:entity/columns", Middlewares: mws(
		EnsureViewConfigurationsEnabled,
		EnsureViewEntityReadable,
		httputils.ValidateAndTransformQuery(columns.AdminGetColumnsParams, httputils.QueryConfig{}),
	)},
}

// Middlewares are the view route middlewares in upstream order
// (configurations, entities, columns).
var Middlewares = append(append(append([]medusahttp.MiddlewareRoute{}, ViewConfigurationRoutesMiddlewares...), EntitiesRoutesMiddlewares...), ColumnRoutesMiddlewares...)
