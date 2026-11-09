// Package region mirrors @medusajs/region.
package region

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/region/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/region/src/models"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations: migrations.Migrations,
	Models:     models.All,
}
