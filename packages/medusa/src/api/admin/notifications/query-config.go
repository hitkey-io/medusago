// Package notifications mirrors api/admin/notifications.
package notifications

import httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"

// DefaultAdminNotificationFields mirrors defaultAdminNotificationFields.
var DefaultAdminNotificationFields = []string{"id", "to", "channel", "template", "data", "trigger_type", "resource_id", "resource_type", "receiver_id", "created_at", "updated_at"}

// RetrieveTransformQueryConfig mirrors retrieveTransformQueryConfig.
var RetrieveTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminNotificationFields, IsList: false, Entity: "notification"}

// ListTransformQueryConfig mirrors listTransformQueryConfig.
var ListTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminNotificationFields, IsList: true, Entity: "notification"}
