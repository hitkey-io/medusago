package utils

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/hitkey-io/medusago/packages/modules/link-modules/src/definitions"
)

// testdata/upstream-link-sql.json is written by scripts/extract-links.mjs
// with the SQL upstream generates for every link table.
func TestCreateSchemaSQLMatchesUpstream(t *testing.T) {
	raw, err := os.ReadFile("testdata/upstream-link-sql.json")
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{}
	if err := json.Unmarshal(raw, &expected); err != nil {
		t.Fatal(err)
	}

	seen := 0
	for _, cfg := range definitions.All {
		if cfg.IsReadOnlyLink || len(cfg.Relationships) != 2 {
			continue
		}
		entity, err := GenerateEntity(cfg, cfg.Relationships[0], cfg.Relationships[1])
		if err != nil {
			t.Fatal(err)
		}
		want, ok := expected[entity.TableName]
		if !ok {
			t.Errorf("no upstream SQL for %s", entity.TableName)
			continue
		}
		if got := entity.CreateSchemaSQL(); got != want {
			t.Errorf("%s:\n got: %q\nwant: %q", entity.TableName, got, want)
		}
		seen++
	}
	if seen != len(expected) {
		t.Errorf("checked %d link tables, upstream has %d", seen, len(expected))
	}
}
