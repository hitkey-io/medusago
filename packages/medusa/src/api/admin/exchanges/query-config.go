// Package exchanges mirrors api/admin/exchanges.
package exchanges

import httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"

// DefaultAdminExchangeFields mirrors defaultAdminExchangeFields.
var DefaultAdminExchangeFields = []string{"id", "order_id", "return_id", "display_id", "order_version", "created_by", "created_at", "updated_at", "canceled_at"}

// DefaultAdminDetailsExchangeFields mirrors defaultAdminDetailsExchangeFields.
var DefaultAdminDetailsExchangeFields = append(append([]string{}, DefaultAdminExchangeFields...), "additional_items.*")

// RetrieveTransformQueryConfig mirrors retrieveTransformQueryConfig.
var RetrieveTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminDetailsExchangeFields, Entity: "order_exchange"}

// ListTransformQueryConfig mirrors listTransformQueryConfig.
var ListTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminExchangeFields, DefaultLimit: 20, IsList: true, Entity: "order_exchange"}
