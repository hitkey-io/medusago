package wishlist

import (
	"testing"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/framework/src/plugins"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func TestApplyPlugin(t *testing.T) {
	cfg := config.DefineConfig(config.InputConfig{
		Plugins: []config.PluginConfig{{Resolve: Name, Options: map[string]any{"max_items": 10}}},
	})
	if err := plugins.Apply(&cfg); err != nil {
		t.Fatal(err)
	}

	var found *config.ModuleConfig
	for i, m := range cfg.Modules {
		if m.Key == "wishlist" {
			found = &cfg.Modules[i]
		}
	}
	if found == nil || found.Options["max_items"] != 10 {
		t.Fatalf("%+v", found)
	}
	if len(utilsmodulessdk.GetCustomLinks()) != 1 {
		t.Fatal("link not registered")
	}
	if !composer.IsRegisteredWorkflow("job-wishlist-cleanup") {
		t.Fatal("job not registered")
	}
	resolved, err := plugins.Resolved(cfg)
	if err != nil || len(resolved) != 1 || len(resolved[0].Routes) != 1 || len(resolved[0].Subscribers) != 1 {
		t.Fatalf("%v %v", resolved, err)
	}
}
