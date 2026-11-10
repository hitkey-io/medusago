// Package configurations mirrors api/admin/views/[entity]/configurations.
package configurations

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	ff "github.com/hitkey-io/medusago/packages/core/utils/src/feature-flags"
	"github.com/hitkey-io/medusago/packages/medusa/src/feature-flags"
)

// EnsureViewConfigurationsEnabled mirrors ensureViewConfigurationsEnabled:
// 404 { type: not_found, message: "Route not found" } unless the
// view_configurations flag is on.
func EnsureViewConfigurationsEnabled(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
	router, _ := req.Scope.Resolve(common.ContainerRegistrationKeys.FEATURE_FLAG_ROUTER).(*ff.FlagRouter)
	if router == nil || !router.IsFeatureEnabled(featureflags.ViewConfigurations.Key) {
		body := dal.NewRecord()
		body.Set("type", "not_found")
		body.Set("message", "Route not found")
		return res.Status(404).JSON(body)
	}
	next(nil)
	return nil
}

// NotPorted answers the view configuration routes when the flag is on:
// the settings module (view configurations, property labels, entity
// discovery) is not ported yet.
func NotPorted(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "View configurations are not supported yet")
}

// Route is the route file.
var Route = medusahttp.Route{GET: NotPorted, POST: NotPorted}

func mw(method, matcher string) medusahttp.MiddlewareRoute {
	return medusahttp.MiddlewareRoute{Methods: []string{method}, Matcher: matcher, Middlewares: []medusahttp.MiddlewareFunction{EnsureViewConfigurationsEnabled}}
}

// Middlewares mirrors the view configuration and property label
// middlewares (the flag guard).
var Middlewares = []medusahttp.MiddlewareRoute{
	mw("GET", "/admin/views/entities"),
	{Methods: []string{"GET", "POST", "DELETE"}, Matcher: "/admin/views/*/configurations*", Middlewares: []medusahttp.MiddlewareFunction{EnsureViewConfigurationsEnabled}},
	{Methods: []string{"GET", "POST"}, Matcher: "/admin/views/*/columns", Middlewares: []medusahttp.MiddlewareFunction{EnsureViewConfigurationsEnabled}},
	mw("GET", "/admin/views/:entity/columns"),
	mw("GET", "/admin/property-labels"),
	mw("POST", "/admin/property-labels"),
	mw("GET", "/admin/property-labels/:id"),
	mw("POST", "/admin/property-labels/:id"),
	mw("DELETE", "/admin/property-labels/:id"),
	mw("POST", "/admin/property-labels/batch"),
}
