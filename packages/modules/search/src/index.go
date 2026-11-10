// Package search mirrors @medusajs/search.
package search

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/search/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/search/src/models"
	"github.com/hitkey-io/medusago/packages/modules/search/src/services"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations:   migrations.Migrations,
	Models:       models.All,
	JoinerConfig: JoinerConfig,
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		return services.NewSearchModuleService(opts)
	},
}
