// Package stores mirrors api/admin/stores.
package stores

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	ff "github.com/hitkey-io/medusago/packages/core/utils/src/feature-flags"
	featureflags "github.com/hitkey-io/medusago/packages/medusa/src/feature-flags"
)

// DefaultAdminStoreFields mirrors defaultAdminStoreFields with the
// translation feature flag disabled.
var DefaultAdminStoreFields = []string{"id", "name", "*supported_currencies", "*supported_currencies.currency", "default_sales_channel_id", "default_region_id", "default_location_id", "metadata", "created_at", "updated_at"}

// DefaultAdminStoreFieldsWithTranslation mirrors defaultAdminStoreFields
// with the translation feature flag enabled (supported locales).
var DefaultAdminStoreFieldsWithTranslation = []string{"id", "name", "*supported_currencies", "*supported_currencies.currency", "*supported_locales", "*supported_locales.locale", "default_sales_channel_id", "default_region_id", "default_location_id", "metadata", "created_at", "updated_at"}

// RetrieveTransformQueryConfig mirrors retrieveTransformQueryConfig.
var RetrieveTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminStoreFields, IsList: false, Entity: "store"}

// ListTransformQueryConfig mirrors listTransformQueryConfig.
var ListTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminStoreFields, IsList: true, Entity: "store"}

// validateAndTransformQuery uses the query config of the translation flag
// state: upstream builds defaultAdminStoreFields when the route files are
// loaded, after the flags.
func validateAndTransformQuery(schema *zod.Schema, qc httputils.QueryConfig) medusahttp.MiddlewareFunction {
	without := httputils.ValidateAndTransformQuery(schema, qc)
	qc.Defaults = DefaultAdminStoreFieldsWithTranslation
	with := httputils.ValidateAndTransformQuery(schema, qc)
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		router, _ := req.Scope.Resolve(common.ContainerRegistrationKeys.FEATURE_FLAG_ROUTER).(*ff.FlagRouter)
		if router != nil && router.IsFeatureEnabled(featureflags.Translation.Key) {
			return with(req, res, next)
		}
		return without(req, res, next)
	}
}
