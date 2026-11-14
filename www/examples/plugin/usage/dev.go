package usage

import (
	"os"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	wishlistmodule "github.com/hitkey-io/medusago/www/examples/plugin/src/modules/wishlist"
)

// DevConfig is the configuration of the development application of the
// plugin, in its dev directory.
func DevConfig() config.InputConfig {
	return config.InputConfig{
		ProjectConfig: config.ProjectConfig{
			DatabaseURL: os.Getenv("DATABASE_URL"),
		},
		// region: dev-module
		Modules: []config.ModuleConfig{
			{
				// the path of the module's directory, relative to the dev directory
				Resolve: "../src/modules/wishlist",
				Module:  &wishlistmodule.Module.ModuleExports,
			},
		},
		// endregion: dev-module
	}
}
