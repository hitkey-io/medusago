package main

// region: config-imports
import (
	"os"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/www/examples/modules/blog"
)

// endregion: config-imports

// region: config
// Config is the configuration of the application.
func Config() config.InputConfig {
	return config.InputConfig{
		ProjectConfig: config.ProjectConfig{
			DatabaseURL: os.Getenv("DATABASE_URL"),
		},
		Modules: []config.ModuleConfig{
			{
				// the path of the module directory: db:generate writes
				// the migrations there
				Resolve: "./src/modules/blog",
				Module:  &blog.Module.ModuleExports,
				Options: map[string]any{
					"defaultTags":    []string{"news", "release"},
					"maxTitleLength": 80,
				},
			},
		},
	}
}

// endregion: config
