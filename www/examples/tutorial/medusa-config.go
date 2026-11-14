package main

import (
	"os"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/www/examples/tutorial/src/modules/brand"
)

// medusaConfig is the configuration of the application.
func medusaConfig() config.InputConfig {
	return config.InputConfig{
		ProjectConfig: config.ProjectConfig{
			DatabaseURL: os.Getenv("DATABASE_URL"),
		},
		// region: modules
		Modules: []config.ModuleConfig{
			{
				Resolve: "./src/modules/brand",
				Module:  &brand.Module.ModuleExports,
			},
		},
		// endregion: modules
	}
}
