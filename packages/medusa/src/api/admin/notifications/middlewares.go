package notifications

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// AdminNotificationRoutesMiddlewares mirrors adminNotificationRoutesMiddlewares.
var AdminNotificationRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/notifications", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetNotificationsParams, ListTransformQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/notifications/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetNotificationParams, RetrieveTransformQueryConfig)}},
}
