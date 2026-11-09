// Package users mirrors api/admin/users.
package users

import httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"

// DefaultAdminUserFields mirrors defaultAdminUserFields.
var DefaultAdminUserFields = []string{"id", "first_name", "last_name", "email", "avatar_url", "metadata", "created_at", "updated_at", "deleted_at"}

// RetrieveTransformQueryConfig mirrors retrieveTransformQueryConfig.
var RetrieveTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminUserFields, IsList: false, Entity: "user"}

// ListTransformQueryConfig mirrors listTransformQueryConfig.
var ListTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminUserFields, IsList: true, Entity: "user"}
