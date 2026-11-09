// Package user mirrors @medusajs/user.
package user

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/user/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/user/src/models"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations: migrations.Migrations,
	Models:     models.All,
}
