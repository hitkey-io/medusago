// Package stores mirrors api/admin/stores.
package stores

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
)

// DefaultAdminStoreFields mirrors defaultAdminStoreFields.
var DefaultAdminStoreFields = []string{"id", "name", "*supported_currencies", "*supported_currencies.currency", "*supported_locales", "*supported_locales.locale", "default_sales_channel_id", "default_region_id", "default_location_id", "metadata", "created_at", "updated_at"}

// RetrieveTransformQueryConfig mirrors retrieveTransformQueryConfig.
var RetrieveTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminStoreFields, IsList: false, Entity: "store"}

// ListTransformQueryConfig mirrors listTransformQueryConfig.
var ListTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminStoreFields, IsList: true, Entity: "store"}

// validateAndTransformQuery validates the query with the given config.
func validateAndTransformQuery(schema *zod.Schema, qc httputils.QueryConfig) medusahttp.MiddlewareFunction {
	return httputils.ValidateAndTransformQuery(schema, qc)
}
