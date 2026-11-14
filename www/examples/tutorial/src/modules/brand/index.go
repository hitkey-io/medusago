// Package brand is the Brand Module: a `brand` table and a service to
// manage it.
package brand

import (
	typesmodulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/www/examples/tutorial/src/modules/brand/migrations"
)

// region: definition
// ModuleName is the name of the module, and the key its service is
// registered under in the container.
const ModuleName = "brand"

// Module is the module's definition.
var Module = utilsmodulessdk.Module(ModuleName, utilsmodulessdk.ModuleDefinition{
	Models:     []*dml.Entity{Brand},
	Migrations: migrations.Migrations, // leave this line out until you generate the migrations
	Service: func(_ typesmodulessdk.ServiceOptions, base *utilsmodulessdk.MedusaService) (any, error) {
		return &BrandModuleService{MedusaService: base}, nil
	},
})

// endregion: definition
