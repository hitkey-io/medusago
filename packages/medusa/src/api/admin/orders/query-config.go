// Package orders mirrors api/admin/orders.
package orders

import httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"

// DefaultAdminOrderFields mirrors defaultAdminOrderFields.
var DefaultAdminOrderFields = []string{"id", "display_id", "custom_display_id", "status", "version", "summary", "total", "metadata", "locale", "created_at", "updated_at"}

// ListTransformQueryConfig mirrors listTransformQueryConfig.
var ListTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminOrderFields, DefaultLimit: 20, IsList: true, Entity: "order"}
