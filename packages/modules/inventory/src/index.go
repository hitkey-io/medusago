// Package inventory mirrors @medusajs/inventory.
package inventory

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/inventory/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/inventory/src/models"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations: migrations.Migrations,
	Models:     models.All,
}
