// Package inventory mirrors @medusajs/inventory.
package inventory

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/inventory/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/inventory/src/models"
	"github.com/hitkey-io/medusago/packages/modules/inventory/src/services"
	_ "github.com/hitkey-io/medusago/packages/modules/inventory/src/utils"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations:   migrations.Migrations,
	Models:       models.All,
	JoinerConfig: JoinerConfig,
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		return services.NewInventoryModuleService(opts, JoinerConfig)
	},
}
