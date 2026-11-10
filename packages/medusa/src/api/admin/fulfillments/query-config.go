// Package fulfillments mirrors api/admin/fulfillments.
package fulfillments

import httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"

// DefaultAdminFulfillmentsFields mirrors defaultAdminFulfillmentsFields.
var DefaultAdminFulfillmentsFields = []string{"id", "location_id", "packed_at", "shipped_at", "marked_shipped_by", "created_by", "delivered_at",
	"canceled_at", "data", "provider_id", "shipping_option_id", "metadata", "order", "created_at", "updated_at", "deleted_at", "*delivery_address",
	"*items", "*labels"}

// RetrieveTransformQueryConfig mirrors retrieveTransformQueryConfig.
var RetrieveTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminFulfillmentsFields, Entity: "fulfillment"}
