// Package loyalty mirrors the loyalty module of @medusajs/loyalty-plugin
// (src/modules/loyalty).
package loyalty

import (
	typesmodulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/loyalty/migrations"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/loyalty/models"
)

// ServiceName is the module key (PluginModule.LOYALTY).
const ServiceName = "loyalty"

// Module mirrors index.ts: Module("loyalty", { service: LoyaltyModuleService }).
var Module = utilsmodulessdk.Module(ServiceName, utilsmodulessdk.ModuleDefinition{
	Models:     models.All,
	Migrations: migrations.Migrations,
	Service: func(opts typesmodulessdk.ServiceOptions, base *utilsmodulessdk.MedusaService) (any, error) {
		options := opts.Options
		if options == nil {
			options = map[string]any{}
		}
		return &LoyaltyModuleService{MedusaService: base, options: options}, nil
	},
})
