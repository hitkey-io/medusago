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

// EnsureViewEntityReadable mirrors ensureViewEntityReadable: a no-op when
// RBAC is not installed (RBAC is not ported).
func EnsureViewEntityReadable(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
	next(nil)
	return nil
}

// SystemDefaultViewConfigurationMessage mirrors
// SYSTEM_DEFAULT_VIEW_CONFIGURATION_MESSAGE.
const SystemDefaultViewConfigurationMessage = "Only super admins can manage system default view configurations"

// RequireSuperAdmin mirrors requireSuperAdmin(when, message): a no-op when
// RBAC is not installed (RBAC is not ported).
func RequireSuperAdmin(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
	next(nil)
	return nil
}

// NotPorted answers the view routes that are not ported yet (entity
// discovery: /admin/views/entities and /admin/views/:entity/columns).
func NotPorted(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "View configurations are not supported yet")
}
