package utils_test

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/modules/caching/src/loaders"
	"github.com/hitkey-io/medusago/packages/modules/caching/src/utils"
)

// Vectors captured from upstream (testdata/upstream-vectors.json, built by
// running the upstream CacheInvalidationParser, fast-json-stable-stringify
// and xxhash-wasm over query results of the http-diff seed; the parser
// samples are checked in packages/medusa/src/loaders, over the schema of
// the default modules).

type vectors struct {
	Hashes []struct {
		Input string `json:"input"`
		Hash  string `json:"hash"`
	} `json:"hashes"`
	Stringify []struct {
		Input  json.RawMessage `json:"input"`
		Output string          `json:"output"`
	} `json:"stringify"`
	Samples []struct {
		Input    json.RawMessage `json:"input"`
		Entities []struct {
			Type      string `json:"type"`
			ID        any    `json:"id"`
			IsInArray bool   `json:"isInArray"`
		} `json:"entities"`
		Tags []string `json:"tags"`
	} `json:"samples"`
	Events []struct {
		Name string          `json:"name"`
		Data json.RawMessage `json:"data"`
		Tags []string        `json:"tags"`
	} `json:"events"`
}

func load(t *testing.T) vectors {
	t.Helper()
	raw, err := os.ReadFile("../testdata/upstream-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v vectors
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func decode(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	v, err := dal.DecodeOrderedJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestHasherMatchesUpstream(t *testing.T) {
	for _, h := range load(t).Hashes {
		if got := loaders.Hasher(h.Input); got != h.Hash {
			t.Errorf("%q: got %s, want %s", h.Input, got, h.Hash)
		}
	}
}

func TestStableStringifyMatchesUpstream(t *testing.T) {
	for _, s := range load(t).Stringify {
		got, err := caching.StableStringify(decode(t, s.Input))
		if err != nil {
			t.Fatal(err)
		}
		if got != s.Output {
			t.Errorf("got  %s\nwant %s", got, s.Output)
		}
	}
}

type recordingClearer struct{ tags []string }

func (r *recordingClearer) Clear(_ context.Context, o caching.ClearOptions) error {
	r.tags = append(r.tags, o.Tags...)
	return nil
}

func TestEventInvalidationTagsMatchUpstream(t *testing.T) {
	for _, ev := range load(t).Events {
		rc := &recordingClearer{}
		s := utils.NewDefaultCacheStrategy(rc, loaders.Hasher, nil)
		s.OnApplicationStart(nil, nil, nil)
		s.HandleEvent(context.Background(), ev.Name, decode(t, ev.Data))
		if !reflect.DeepEqual(rc.tags, ev.Tags) {
			t.Errorf("%s: got %v, want %v", ev.Name, rc.tags, ev.Tags)
		}
	}
}
