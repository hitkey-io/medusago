// Package loyalty shows how a project enables the loyalty plugin (gift cards and
// store credit).
package loyalty

import (
	"os"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"

	// region: import
	// Importing the package registers the plugin with the framework.
	_ "github.com/hitkey-io/medusago/packages/plugins/loyalty/src"
	// endregion: import
)

// MedusaConfig is the project's medusa-config.go.
func MedusaConfig() config.InputConfig {
	return config.InputConfig{
		ProjectConfig: config.ProjectConfig{
			DatabaseURL: os.Getenv("DATABASE_URL"),
		},
		// region: plugin
		Plugins: []config.PluginConfig{
			{Resolve: "@medusajs/loyalty-plugin"},
		},
		// endregion: plugin
	}
}
