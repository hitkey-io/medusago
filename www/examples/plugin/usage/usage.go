// Package usage shows how an application uses the plugin.
package usage

import (
	"os"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"

	// Importing the package registers the plugin.
	_ "github.com/hitkey-io/medusago/www/examples/plugin"
)

// MedusaConfig is the project's medusa-config.go.
func MedusaConfig() config.InputConfig {
	return config.InputConfig{
		ProjectConfig: config.ProjectConfig{
			DatabaseURL: os.Getenv("DATABASE_URL"),
		},
		// region: config
		Plugins: []config.PluginConfig{
			{
				Resolve: "github.com/acme/medusa-plugin-wishlist",
				Options: map[string]any{
					"max_items": 50,
				},
			},
		},
		// endregion: config
	}
}
