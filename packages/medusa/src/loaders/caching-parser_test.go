package loaders_test

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
	"github.com/hitkey-io/medusago/packages/modules/caching/src/utils"
)

// The cache tags computed from query results over the merged schema of the
// default modules and links match upstream's CacheInvalidationParser
// (packages/modules/caching/src/testdata/upstream-vectors.json).
func TestCacheInvalidationParserMatchesUpstream(t *testing.T) {
	raw, err := os.ReadFile("../../../modules/caching/src/testdata/upstream-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Samples []struct {
			Input    json.RawMessage `json:"input"`
			Entities []struct {
				Type      string `json:"type"`
				ID        any    `json:"id"`
				IsInArray bool   `json:"isInArray"`
			} `json:"entities"`
			Tags []string `json:"tags"`
		} `json:"samples"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	qc, err := loaders.BuildQueryConfig(config.DefineConfig(config.InputConfig{}))
	if err != nil {
		t.Fatal(err)
	}
	p := utils.NewCacheInvalidationParser(qc.Schema, qc.JoinerConfigs)
	for i, s := range v.Samples {
		input, err := dal.DecodeOrderedJSON(s.Input)
		if err != nil {
			t.Fatal(err)
		}
		entities := p.ParseObjectForEntities(input, "", false)
		var got, want []string
		for _, e := range entities {
			got = append(got, fmt.Sprintf("%s:%v:%v", e.Type, e.ID, e.IsInArray))
		}
		for _, e := range s.Entities {
			want = append(want, fmt.Sprintf("%s:%v:%v", e.Type, e.ID, e.IsInArray))
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("sample %d entities\ngot  %v\nwant %v", i, got, want)
		}
		tags := []string{}
		seen := map[string]bool{}
		for _, ev := range p.BuildInvalidationEvents(entities, "") {
			for _, k := range ev.CacheKeys {
				if !seen[k] {
					seen[k] = true
					tags = append(tags, k)
				}
			}
		}
		if !reflect.DeepEqual(tags, s.Tags) {
			t.Errorf("sample %d tags\ngot  %v\nwant %v", i, tags, s.Tags)
		}
	}
}
