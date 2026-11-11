package modulessdk_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
	"github.com/hitkey-io/medusago/packages/modules/link-modules/src/definitions"
	"github.com/hitkey-io/medusago/packages/modules/link-modules/src/definitions/readonly"
)

func loadedModules(cfg config.ConfigModule) []modulessdk.LoadedModule {
	var out []modulessdk.LoadedModule
	for _, mc := range cfg.Modules {
		if mc.Disable {
			continue
		}
		lm := modulessdk.LoadedModule{Key: mc.Key}
		if exp, ok := modules.Registry[mc.Resolve]; ok {
			lm.JoinerConfig = exp.JoinerConfig
		}
		out = append(out, lm)
	}
	return out
}

func withoutSchema(t *testing.T, configs []*joiner.ModuleJoinerConfig) []string {
	t.Helper()
	var out []string
	for _, c := range configs {
		c = c.Clone()
		c.Schema = ""
		b, err := json.Marshal(c)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, string(b))
	}
	return out
}

func coreLinkDefinitions() []*joiner.ModuleJoinerConfig {
	return append(append([]*joiner.ModuleJoinerConfig{}, definitions.All...), readonly.All...)
}

// The joiner configs and relation map built at runtime must equal what
// upstream loads with the default configuration (extracted from a running
// upstream application).
func TestBuildQueryConfigMatchesUpstream(t *testing.T) {
	cfg := config.DefineConfig(config.InputConfig{})
	qc, err := modulessdk.BuildQueryConfig(loadedModules(cfg), coreLinkDefinitions(), nil)
	if err != nil {
		t.Fatal(err)
	}
	got := withoutSchema(t, qc.JoinerConfigs)
	want := withoutSchema(t, modulessdk.DefaultJoinerConfigs)
	if len(got) != len(want) {
		t.Errorf("got %d joiner configs, want %d", len(got), len(want))
	}
	for i := 0; i < len(got) && i < len(want); i++ {
		if got[i] != want[i] {
			t.Errorf("joiner config %d differs\nwant: %s\ngot:  %s", i, want[i], got[i])
		}
	}
	compareRelationMaps(t, qc.RelationMap, modulessdk.DefaultRelationMap)
}

func compareRelationMaps(t *testing.T, got, want map[string]map[string]string) {
	t.Helper()
	for entity, props := range want {
		if !reflect.DeepEqual(got[entity], props) {
			t.Errorf("relation map %s:\nwant %v\ngot  %v", entity, props, got[entity])
		}
	}
	for entity := range got {
		if _, ok := want[entity]; !ok {
			t.Errorf("relation map has extra entity %s: %v", entity, got[entity])
		}
	}
}

func TestBuildQueryConfigWithTranslationMatchesUpstream(t *testing.T) {
	cfg := config.DefineConfig(config.InputConfig{FeatureFlags: map[string]any{"translation": true}})
	qc, err := modulessdk.BuildQueryConfig(loadedModules(cfg), coreLinkDefinitions(), nil)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, s := range withoutSchema(t, qc.JoinerConfigs) {
		got[s] = true
	}
	all := append(append([]*joiner.ModuleJoinerConfig{}, modulessdk.DefaultJoinerConfigs...), modulessdk.TranslationJoinerConfigs...)
	wantList := withoutSchema(t, all)
	for _, s := range wantList {
		if !got[s] {
			t.Errorf("missing joiner config %s", s)
		}
	}
	if len(got) != len(wantList) {
		t.Errorf("got %d joiner configs, want %d", len(got), len(wantList))
	}
	want := map[string]map[string]string{}
	for e, p := range modulessdk.DefaultRelationMap {
		want[e] = map[string]string{}
		for k, v := range p {
			want[e][k] = v
		}
	}
	for e, p := range modulessdk.TranslationRelationMap {
		if want[e] == nil {
			want[e] = map[string]string{}
		}
		for k, v := range p {
			want[e][k] = v
		}
	}
	compareRelationMaps(t, qc.RelationMap, want)
}
