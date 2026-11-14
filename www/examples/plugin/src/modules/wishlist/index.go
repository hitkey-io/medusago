// Package wishlist is the Wishlist Module of the plugin.
package wishlist

import (
	"github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/www/examples/plugin/src/modules/wishlist/migrations"
)

// ModuleName is the name of the module.
const ModuleName = "wishlist"

// WishlistModuleService is the main service of the module.
type WishlistModuleService struct {
	*utilsmodulessdk.MedusaService
	// MaxItems is the `max_items` option of the plugin.
	MaxItems int
}

// region: module
// Module is the module's definition. The options of the plugin are passed
// to the module in the options of its service.
var Module = utilsmodulessdk.Module(ModuleName, utilsmodulessdk.ModuleDefinition{
	Models:     []*dml.Entity{Wishlist},
	Migrations: migrations.Migrations,
	Service: func(opts modulessdk.ServiceOptions, base *utilsmodulessdk.MedusaService) (any, error) {
		service := &WishlistModuleService{MedusaService: base, MaxItems: 100}
		if max, ok := opts.Options["max_items"].(int); ok {
			service.MaxItems = max
		}
		return service, nil
	},
})

// endregion: module
