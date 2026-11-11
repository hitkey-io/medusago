package modules

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
)

func metaJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	// normalize default values (raw JSON) through a decode/encode round trip
	var x any
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := dec.Decode(&x); err != nil {
		t.Fatal(err)
	}
	b, _ = json.MarshalIndent(x, "", " ")
	return string(b)
}

// The metadata derived from the DML definitions must equal the metadata
// upstream resolves through MikroORM, captured with the generated models.
func TestDerivedMetaMatchesUpstream(t *testing.T) {
	total := 0
	for pkg, models := range allModels() {
		derived, err := dml.DeriveMeta(models)
		if err != nil {
			t.Errorf("%s: %v", pkg, err)
			continue
		}
		for i, e := range models {
			total++
			want := metaJSON(t, e.Meta)
			got := metaJSON(t, derived[i])
			if want != got {
				t.Errorf("%s %s: derived meta differs", pkg, e.Name)
				for j := 0; j < len(e.Meta.Properties) || j < len(derived[i].Properties); j++ {
					var w, g any
					if j < len(e.Meta.Properties) {
						w = e.Meta.Properties[j]
					}
					if j < len(derived[i].Properties) {
						g = derived[i].Properties[j]
					}
					wj, gj := compactJSON(t, w), compactJSON(t, g)
					if wj != gj {
						t.Errorf("  want %s\n  got  %s", wj, gj)
					}
				}
				if a, b := compactJSON(t, e.Meta.PrimaryKey), compactJSON(t, derived[i].PrimaryKey); a != b {
					t.Errorf("  pks %s vs %s", a, b)
				}
			}
		}
	}
	if total < 120 {
		t.Fatalf("only %d models checked", total)
	}
}

func compactJSON(t *testing.T, v any) string {
	var buf bytes.Buffer
	_ = json.Compact(&buf, []byte(metaJSON(t, v)))
	return buf.String()
}
