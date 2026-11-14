package configexamples

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/framework/src/container"
	featureflags "github.com/hitkey-io/medusago/packages/medusa/src/feature-flags"
)

// FeatureFlagsExample enables the index engine and the translation flags.
func FeatureFlagsExample() config.InputConfig {
	// region: feature-flags
	return config.InputConfig{
		FeatureFlags: map[string]any{
			"index_engine": true,
			"translation":  true,
		},
	}
	// endregion: feature-flags
}

// IndexEngineEnabled reports whether the index engine flag is on. It accepts
// the container of a script, a subscriber or a request (req.Scope).
func IndexEngineEnabled(c *container.MedusaContainer) bool {
	// region: check-flag
	return featureflags.IsFeatureEnabled(c, featureflags.IndexEngine.Key)
	// endregion: check-flag
}

// IndexEngineConfig enables the index engine: the flag, and the Index Module.
func IndexEngineConfig() config.InputConfig {
	// region: index-engine
	return config.InputConfig{
		FeatureFlags: map[string]any{
			"index_engine": true,
		},
		Modules: []config.ModuleConfig{
			{Key: "index", Resolve: "@medusajs/medusa/index-module"},
		},
	}
	// endregion: index-engine
}
