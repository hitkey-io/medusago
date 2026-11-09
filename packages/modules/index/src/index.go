// Package index mirrors @medusajs/index.
package index

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/index/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/index/src/models"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations: migrations.Migrations,
	Models:     models.All,
}
