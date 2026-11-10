package configurations

import (
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// DefaultViewConfigurationFields mirrors defaultViewConfigurationFields.
var DefaultViewConfigurationFields = []string{
	"id",
	"entity",
	"name",
	"user_id",
	"is_system_default",
	"configuration",
	"created_at",
	"updated_at",
}

// Query configs.
var (
	RetrieveViewConfigurationList = httputils.QueryConfig{Defaults: DefaultViewConfigurationFields, IsList: true}
	RetrieveViewConfiguration     = httputils.QueryConfig{Defaults: DefaultViewConfigurationFields}
)
