// Package saleschannels mirrors api/admin/sales-channels.
package saleschannels

import httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"

// DefaultAdminSalesChannelFields mirrors defaultAdminSalesChannelFields.
var DefaultAdminSalesChannelFields = []string{"id", "name", "description", "is_disabled", "created_at", "updated_at", "deleted_at", "metadata"}

// RetrieveTransformQueryConfig mirrors retrieveTransformQueryConfig.
var RetrieveTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminSalesChannelFields, IsList: false, Entity: "sales_channel"}

// ListTransformQueryConfig mirrors listTransformQueryConfig.
var ListTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminSalesChannelFields, IsList: true, Entity: "sales_channel"}
