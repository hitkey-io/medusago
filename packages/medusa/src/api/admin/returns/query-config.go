// Package returns mirrors api/admin/returns.
package returns

import httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"

// DefaultAdminReturnFields mirrors defaultAdminReturnFields.
var DefaultAdminReturnFields = []string{"id", "order_id", "exchange_id", "claim_id", "display_id", "location_id", "order_version", "status", "metadata",
	"no_notification", "refund_amount", "created_by", "created_at", "updated_at", "canceled_at", "requested_at", "received_at"}

// DefaultAdminDetailsReturnFields mirrors defaultAdminDetailsReturnFields.
var DefaultAdminDetailsReturnFields = append(append([]string{}, DefaultAdminReturnFields...), "items.*", "items.reason.*")

// RetrieveTransformQueryConfig mirrors retrieveTransformQueryConfig.
var RetrieveTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminDetailsReturnFields, Entity: "return"}

// ListTransformQueryConfig mirrors listTransformQueryConfig.
var ListTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminReturnFields, DefaultLimit: 20, IsList: true, Entity: "return"}
