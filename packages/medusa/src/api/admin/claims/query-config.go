// Package claims mirrors api/admin/claims.
package claims

import httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"

// DefaultAdminClaimFields mirrors defaultAdminClaimFields.
var DefaultAdminClaimFields = []string{"id", "type", "order_id", "return_id", "display_id", "order_version", "refund_amount", "created_by", "created_at", "updated_at", "canceled_at"}

// DefaultAdminDetailsClaimFields mirrors defaultAdminDetailsClaimFields.
var DefaultAdminDetailsClaimFields = append(append([]string{}, DefaultAdminClaimFields...), "additional_items.*", "claim_items.*", "claim_items.reason.*")

// RetrieveTransformQueryConfig mirrors retrieveTransformQueryConfig.
var RetrieveTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminDetailsClaimFields, Entity: "order_claim"}

// ListTransformQueryConfig mirrors listTransformQueryConfig.
var ListTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminClaimFields, DefaultLimit: 20, IsList: true, Entity: "order_claim"}
