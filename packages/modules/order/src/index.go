// Package order mirrors @medusajs/order.
package order

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/order/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/order/src/models"
	"github.com/hitkey-io/medusago/packages/modules/order/src/services"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations:   migrations.Migrations,
	Models:       models.All,
	JoinerConfig: JoinerConfig,
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		return services.NewOrderModuleService(opts, JoinerConfig)
	},
}
