package configexamples

import (
	"os"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
)

// ProductionModules replaces the development defaults of the Event, Workflow
// Engine, Locking and Caching modules with their Redis implementations.
func ProductionModules() []config.ModuleConfig {
	// region: production-modules
	redisURL := os.Getenv("REDIS_URL")

	return []config.ModuleConfig{
		{
			Resolve: "@medusajs/medusa/event-bus-redis",
			Options: map[string]any{"redisUrl": redisURL},
		},
		{
			Resolve: "@medusajs/medusa/workflow-engine-redis",
			Options: map[string]any{
				"redis": map[string]any{"redisUrl": redisURL},
			},
		},
		{
			Key:     "locking",
			Resolve: "@medusajs/medusa/locking",
			Providers: []config.ModuleProviderConfig{
				{
					Resolve:   "@medusajs/medusa/locking-redis",
					ID:        "locking-redis",
					IsDefault: true,
					Options:   map[string]any{"redisUrl": redisURL},
				},
			},
		},
		{
			Key:     "caching",
			Resolve: "@medusajs/medusa/caching",
			Providers: []config.ModuleProviderConfig{
				{
					Resolve:   "@medusajs/medusa/caching-redis",
					ID:        "caching-redis",
					IsDefault: true,
					Options:   map[string]any{"redisUrl": redisURL},
				},
			},
		},
	}
	// endregion: production-modules
}

// S3FileModule replaces the local file provider with S3.
func S3FileModule() config.ModuleConfig {
	// region: s3
	return config.ModuleConfig{
		Key:     "file",
		Resolve: "@medusajs/medusa/file",
		Providers: []config.ModuleProviderConfig{
			{
				Resolve:   "@medusajs/medusa/file-s3",
				ID:        "s3",
				IsDefault: true,
				Options: map[string]any{
					"file_url":          os.Getenv("S3_FILE_URL"),
					"access_key_id":     os.Getenv("S3_ACCESS_KEY_ID"),
					"secret_access_key": os.Getenv("S3_SECRET_ACCESS_KEY"),
					"region":            os.Getenv("S3_REGION"),
					"bucket":            os.Getenv("S3_BUCKET"),
					"endpoint":          os.Getenv("S3_ENDPOINT"),
				},
			},
		},
	}
	// endregion: s3
}

// ModulesInConfig passes the modules to the configuration.
func ModulesInConfig() config.InputConfig {
	// region: modules
	return config.InputConfig{
		Modules: append(ProductionModules(), S3FileModule()),
	}
	// endregion: modules
}

// PluginsExample enables a plugin that is linked into the binary.
func PluginsExample() config.InputConfig {
	// region: plugins
	return config.InputConfig{
		Plugins: []config.PluginConfig{
			{Resolve: "@medusajs/loyalty-plugin", Options: map[string]any{}},
		},
	}
	// endregion: plugins
}
