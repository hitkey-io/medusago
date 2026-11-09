// Package tax mirrors @medusajs/tax.
package tax

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/tax/src/loaders"
	"github.com/hitkey-io/medusago/packages/modules/tax/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/tax/src/models"
	"github.com/hitkey-io/medusago/packages/modules/tax/src/services"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations:   migrations.Migrations,
	Models:       models.All,
	JoinerConfig: JoinerConfig,
	Loaders:      []modulessdk.Loader{loaders.Providers},
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		return services.NewTaxModuleService(opts, JoinerConfig)
	},
}
