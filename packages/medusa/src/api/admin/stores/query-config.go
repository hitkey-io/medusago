// Package stores mirrors api/admin/stores.
package stores

import httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"

// DefaultAdminStoreFields mirrors defaultAdminStoreFields (translation
// feature flag disabled).
var DefaultAdminStoreFields = []string{"id", "name", "*supported_currencies", "*supported_currencies.currency", "default_sales_channel_id", "default_region_id", "default_location_id", "metadata", "created_at", "updated_at"}

// RetrieveTransformQueryConfig mirrors retrieveTransformQueryConfig.
var RetrieveTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminStoreFields, IsList: false, Entity: "store"}

// ListTransformQueryConfig mirrors listTransformQueryConfig.
var ListTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminStoreFields, IsList: true, Entity: "store"}
