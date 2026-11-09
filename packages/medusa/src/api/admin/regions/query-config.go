// Package regions mirrors api/admin/regions.
package regions

import httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"

// DefaultAdminRegionFields mirrors defaultAdminRegionFields.
var DefaultAdminRegionFields = []string{"id", "name", "currency_code", "created_at", "updated_at", "deleted_at", "automatic_taxes", "metadata", "*countries"}

// RetrieveTransformQueryConfig mirrors retrieveTransformQueryConfig.
var RetrieveTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminRegionFields, IsList: false, Entity: "region"}

// ListTransformQueryConfig mirrors listTransformQueryConfig.
var ListTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminRegionFields, DefaultLimit: 20, IsList: true, Entity: "region"}
