package utils

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
)

// testdata/upstream-link-extra-types.json holds the create SQL upstream
// (generateEntity + MikroORM) produces for a link with one extra column of
// every LinkModulesExtraFields type, nullable or with a default, and with
// column options.
func TestExtraColumnTypesMatchUpstream(t *testing.T) {
	raw, err := os.ReadFile("testdata/upstream-link-extra-types.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []struct {
		Field json.RawMessage `json:"field"`
		SQL   string          `json:"sql"`
		Error string          `json:"error"`
	}
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	for _, v := range vectors {
		cfg := joiner.MustParseModuleJoinerConfig(`{
			"serviceName": "TestLink",
			"isLink": true,
			"databaseConfig": {"tableName": "brand_test_link", "extraFields": {"value": ` + string(v.Field) + `}},
			"relationships": [
				{"serviceName": "brand", "primaryKey": "id", "foreignKey": "brand_id", "alias": "brand"},
				{"serviceName": "sales_channel", "primaryKey": "id", "foreignKey": "sales_channel_id", "alias": "sales_channel"}
			]
		}`)
		entity, err := GenerateEntity(cfg, cfg.Relationships[0], cfg.Relationships[1])
		if v.Error != "" {
			if err == nil {
				t.Errorf("%s: expected an error like upstream (%s)", v.Field, v.Error)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", v.Field, err)
			continue
		}
		if got := entity.CreateSchemaSQL(); got != v.SQL {
			t.Errorf("%s:\n got: %q\nwant: %q", v.Field, got, v.SQL)
		}
	}
}
