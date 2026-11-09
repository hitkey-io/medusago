// Package promotion mirrors @medusajs/promotion.
package promotion

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/promotion/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/promotion/src/models"
	"github.com/hitkey-io/medusago/packages/modules/promotion/src/services"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations:   migrations.Migrations,
	Models:       models.All,
	JoinerConfig: JoinerConfig,
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		return services.NewPromotionModuleService(opts, JoinerConfig)
	},
}
