// Package configurations mirrors api/admin/layouts/configurations.
package configurations

import httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"

// DefaultLayoutConfigurationFields mirrors defaultLayoutConfigurationFields.
var DefaultLayoutConfigurationFields = []string{"id", "zone", "user_id", "is_system_default", "configuration", "created_at", "updated_at"}

// RetrieveLayoutConfigurationList mirrors retrieveLayoutConfigurationList.
var RetrieveLayoutConfigurationList = httputils.QueryConfig{Defaults: DefaultLayoutConfigurationFields, IsList: true}
