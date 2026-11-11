package modules

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal/mikro-orm/schema"
	"github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
)

// TestGenerateMigrationsMatchUpstream compares what `medusa db:generate`
// generates from an empty snapshot for the models of every default module
// (the migration statements and the snapshot file) with upstream
// (integration-tests/db-generate/golden/modules.json, written by
// extract-module-vectors.mjs).
func TestGenerateMigrationsMatchUpstream(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "integration-tests", "db-generate", "golden", "modules.json"))
	if err != nil {
		t.Fatal(err)
	}
	var vectors map[string]struct {
		SnapshotName string   `json:"snapshotName"`
		Snapshot     string   `json:"snapshot"`
		Up           []string `json:"up"`
		Down         []string `json:"down"`
	}
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	for name, v := range vectors {
		t.Run(name, func(t *testing.T) {
			exp, ok := Registry["@medusajs/medusa/"+name]
			if !ok && name == "index" {
				exp, ok = Registry["@medusajs/medusa/index-module"]
			}
			if !ok {
				t.Fatalf("module %s not registered", name)
			}
			up, down, snapshot, err := migrations.SchemaDiff([]byte(schema.EmptySnapshot), exp.Models)
			if err != nil {
				t.Fatal(err)
			}
			if snapshot != v.Snapshot {
				t.Errorf("snapshot differs:\n%s", firstDiff(v.Snapshot, snapshot))
			}
			compareStatements(t, "up", v.Up, up)
			compareStatements(t, "down", v.Down, down)
		})
	}
}

func compareStatements(t *testing.T, label string, want, got []string) {
	t.Helper()
	if strings.Join(want, "\n") == strings.Join(got, "\n") {
		return
	}
	n := len(want)
	if len(got) > n {
		n = len(got)
	}
	for i := 0; i < n; i++ {
		var w, g string
		if i < len(want) {
			w = want[i]
		}
		if i < len(got) {
			g = got[i]
		}
		if w != g {
			t.Errorf("%s statement %d differs:\nwant: %s\ngot:  %s", label, i, w, g)
			return
		}
	}
}

func firstDiff(want, got string) string {
	wl, gl := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(wl) || i < len(gl); i++ {
		var w, g string
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w != g {
			from := i - 5
			if from < 0 {
				from = 0
			}
			ctx := strings.Join(wl[from:min(i+1, len(wl))], "\n")
			return "line " + itoa(i+1) + "\ncontext (want):\n" + ctx + "\nwant: " + w + "\ngot:  " + g
		}
	}
	return ""
}

func itoa(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}
