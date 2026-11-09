// Package stocklocation mirrors @medusajs/stock-location.
package stocklocation

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/stock-location/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/stock-location/src/models"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations:   migrations.Migrations,
	Models:       models.All,
	JoinerConfig: JoinerConfig,
}
