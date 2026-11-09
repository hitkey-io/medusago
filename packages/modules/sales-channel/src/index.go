// Package saleschannel mirrors @medusajs/sales-channel.
package saleschannel

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/sales-channel/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/sales-channel/src/models"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations: migrations.Migrations,
	Models:     models.All,
}
