// Package index mirrors api/admin/index.
package index

import (
	"log/slog"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	ff "github.com/hitkey-io/medusago/packages/core/utils/src/feature-flags"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	featureflags "github.com/hitkey-io/medusago/packages/medusa/src/feature-flags"
)

// IsIndexEnabled reports whether the index module is registered and the
// index_engine flag is on.
func IsIndexEnabled(req *medusahttp.MedusaRequest) bool {
	if req.Scope.Resolve(utilsmodulessdk.INDEX) == nil {
		return false
	}
	router, _ := req.Scope.Resolve(common.ContainerRegistrationKeys.FEATURE_FLAG_ROUTER).(*ff.FlagRouter)
	return router != nil && router.IsFeatureEnabled(featureflags.IndexEngine.Key)
}

// isIndexEnabledMiddleware mirrors isIndexEnabledMiddleware.
func isIndexEnabledMiddleware(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
	if !IsIndexEnabled(req) {
		message := "Trying to access '/admin/index/*' route but the index module is not configured"
		logger, _ := req.Scope.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)
		if logger == nil {
			logger = slog.Default()
		}
		logger.Warn(message)
		next(common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, message))
		return nil
	}
	next(nil)
	return nil
}

// AdminIndexRoutesMiddlewares mirrors adminIndexRoutesMiddlewares
// (authenticate is applied by the API loader; authorize is the Enterprise
// RBAC check, a no-op without the RBAC module).
var AdminIndexRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/index/details", Middlewares: []medusahttp.MiddlewareFunction{isIndexEnabledMiddleware}},
	{Methods: []string{"POST"}, Matcher: "/admin/index/sync", Middlewares: []medusahttp.MiddlewareFunction{isIndexEnabledMiddleware, httputils.ValidateAndTransformBody(httputils.Schema(AdminIndexSyncPayload))}},
}
