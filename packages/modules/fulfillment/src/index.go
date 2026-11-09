// Package fulfillment mirrors @medusajs/fulfillment.
package fulfillment

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/fulfillment/src/loaders"
	"github.com/hitkey-io/medusago/packages/modules/fulfillment/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/fulfillment/src/models"
	"github.com/hitkey-io/medusago/packages/modules/fulfillment/src/services"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations:   migrations.Migrations,
	Models:       models.All,
	JoinerConfig: JoinerConfig,
	Loaders:      []modulessdk.Loader{loaders.Providers},
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		return services.NewFulfillmentModuleService(opts, JoinerConfig)
	},
}
