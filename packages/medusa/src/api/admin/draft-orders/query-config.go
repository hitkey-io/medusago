// Package draftorders mirrors api/admin/draft-orders.
package draftorders

import httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"

// DefaultAdminListOrderFields mirrors defaultAdminListOrderFields.
var DefaultAdminListOrderFields = []string{"id", "display_id", "custom_display_id", "status", "version", "email", "region_id", "*items", "summary",
	"total", "metadata", "created_at", "updated_at"}

// DefaultAdminOrderFields mirrors defaultAdminOrderFields.
var DefaultAdminOrderFields = []string{"id", "display_id", "custom_display_id", "status", "version", "email", "region_id", "*items",
	"*items.tax_lines", "*items.adjustments", "*items.detail", "*items.variant", "*items.variant.product", "*shipping_address",
	"*billing_address", "*shipping_methods", "*shipping_methods.tax_lines", "*shipping_methods.adjustments", "summary", "total",
	"metadata", "created_at", "updated_at"}

// RetrieveTransformQueryConfig mirrors retrieveTransformQueryConfig.
var RetrieveTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminOrderFields, Entity: "order"}

// ListTransformQueryConfig mirrors listTransformQueryConfig.
var ListTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminListOrderFields, DefaultLimit: 20, IsList: true, Entity: "order"}
