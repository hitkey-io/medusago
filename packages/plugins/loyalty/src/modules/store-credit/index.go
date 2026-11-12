// Package storecredit mirrors the store credit module of
// @medusajs/loyalty-plugin (src/modules/store-credit).
package storecredit

import (
	typesmodulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/store-credit/migrations"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/store-credit/models"
)

// ServiceName is the module key (PluginModule.STORE_CREDIT).
const ServiceName = "store_credit"

// Module mirrors index.ts: Module("store_credit", { service: StoreCreditService }).
var Module = utilsmodulessdk.Module(ServiceName, utilsmodulessdk.ModuleDefinition{
	Models:     models.All,
	Migrations: migrations.Migrations,
	Service: func(_ typesmodulessdk.ServiceOptions, base *utilsmodulessdk.MedusaService) (any, error) {
		return &StoreCreditModuleService{MedusaService: base}, nil
	},
})
