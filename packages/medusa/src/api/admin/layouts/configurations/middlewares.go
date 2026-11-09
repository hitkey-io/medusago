package configurations

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// LayoutConfigurationListRoutesMiddlewares mirrors layoutConfigurationListRoutesMiddlewares.
var LayoutConfigurationListRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/layouts/configurations", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetLayoutConfigurationsParams, RetrieveLayoutConfigurationList)}},
}
