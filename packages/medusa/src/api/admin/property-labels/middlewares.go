package propertylabels

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/views/entity_/configurations"
)

func mws(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// AdminPropertyLabelsMiddlewares mirrors adminPropertyLabelsMiddlewares
// (authorize is a no-op without RBAC).
var AdminPropertyLabelsMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/property-labels", Middlewares: mws(
		configurations.EnsureViewConfigurationsEnabled,
		httputils.ValidateAndTransformQuery(AdminPropertyLabelListParams, ListTransformQueryConfig),
	)},
	{Methods: []string{"POST"}, Matcher: "/admin/property-labels", Middlewares: mws(
		configurations.EnsureViewConfigurationsEnabled,
		httputils.ValidateAndTransformBody(httputils.Schema(AdminCreatePropertyLabel)),
		httputils.ValidateAndTransformQuery(AdminPropertyLabelParams, RetrieveTransformQueryConfig),
	)},
	{Methods: []string{"GET"}, Matcher: "/admin/property-labels/:id", Middlewares: mws(
		configurations.EnsureViewConfigurationsEnabled,
		httputils.ValidateAndTransformQuery(AdminPropertyLabelParams, RetrieveTransformQueryConfig),
	)},
	{Methods: []string{"POST"}, Matcher: "/admin/property-labels/:id", Middlewares: mws(
		configurations.EnsureViewConfigurationsEnabled,
		httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdatePropertyLabel)),
		httputils.ValidateAndTransformQuery(AdminPropertyLabelParams, RetrieveTransformQueryConfig),
	)},
	{Methods: []string{"DELETE"}, Matcher: "/admin/property-labels/:id", Middlewares: mws(
		configurations.EnsureViewConfigurationsEnabled,
	)},
	{Methods: []string{"POST"}, Matcher: "/admin/property-labels/batch", Middlewares: mws(
		configurations.EnsureViewConfigurationsEnabled,
		httputils.ValidateAndTransformBody(httputils.Schema(AdminBatchPropertyLabels)),
		httputils.ValidateAndTransformQuery(AdminPropertyLabelParams, RetrieveTransformQueryConfig),
	)},
}
