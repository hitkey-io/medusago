// Package store mirrors @medusajs/store.
package store

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/store/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/store/src/models"
	"github.com/hitkey-io/medusago/packages/modules/store/src/services"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations:   migrations.Migrations,
	Models:       models.All,
	JoinerConfig: JoinerConfig,
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		return services.NewStoreModuleService(opts, JoinerConfig)
	},
}
