// Package product mirrors @medusajs/product.
package product

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/product/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/product/src/models"
	"github.com/hitkey-io/medusago/packages/modules/product/src/services"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations:   migrations.Migrations,
	Models:       models.All,
	JoinerConfig: JoinerConfig,
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		return services.NewProductModuleService(opts, JoinerConfig)
	},
}
