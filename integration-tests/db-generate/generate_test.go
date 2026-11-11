// Package dbgenerate is the differential harness of `medusa db:generate`:
// the model versions of versions/ go through the Go generator one after the
// other, and the generated migrations, the snapshot files and the schema
// they produce once applied are compared with what upstream generated for
// the same versions (golden/, recorded by generate.sh).
package dbgenerate

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	typesmodulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
	migrationscripts "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk/migration-scripts"
	"github.com/hitkey-io/medusago/packages/medusa/src/commands/db"
)

type golden struct {
	Up       []string `json:"up"`
	Down     []string `json:"down"`
	Snapshot string   `json:"snapshot"`
}

func loadGolden(t *testing.T, version string) golden {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("golden", version+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var g golden
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatal(err)
	}
	return g
}

// parseMigration reads the statements of a generated Go migration, with ""
// for the blank lines between them (like the golden statements).
func parseMigration(t *testing.T, path string) (name string, up, down []string) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key := kv.Key.(*ast.Ident).Name
			switch key {
			case "Name":
				name, _ = strconv.Unquote(kv.Value.(*ast.BasicLit).Value)
			case "Up", "Down":
				var out []string
				list := kv.Value.(*ast.CompositeLit)
				for i, e := range list.Elts {
					if i > 0 && fset.Position(e.Pos()).Line-fset.Position(list.Elts[i-1].End()).Line > 1 {
						out = append(out, "")
					}
					s, err := strconv.Unquote(e.(*ast.BasicLit).Value)
					if err != nil {
						t.Fatal(err)
					}
					out = append(out, s)
				}
				if key == "Up" {
					up = out
				} else {
					down = out
				}
			}
		}
		return false
	})
	return name, up, down
}

func compareStatements(t *testing.T, label string, want, got []string) {
	t.Helper()
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
		if w != g || (i >= len(want)) != (i >= len(got)) {
			t.Errorf("%s statement %d differs:\nwant: %q\ngot:  %q", label, i, w, g)
			return
		}
	}
}

func withoutBlanks(stmts []string) []string {
	var out []string
	for _, s := range stmts {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// setupDB creates <prefix>gen_go from TEMPLATE (medusa_ref, the core
// modules migrated by upstream) when MEDUSA_TEST_PG_ADMIN_URL is set.
func setupDB(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	admin := os.Getenv("MEDUSA_TEST_PG_ADMIN_URL")
	if admin == "" {
		return nil, ""
	}
	name := os.Getenv("MEDUSA_TEST_DB_PREFIX") + "gen_go"
	template := os.Getenv("MEDUSA_TEST_GEN_TEMPLATE_DB")
	if template == "" {
		template = "medusa_ref"
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{"drop database if exists " + name + " with (force)", "create database " + name + " template " + template} {
		if _, err := conn.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	conn.Close(ctx)
	u, _ := url.Parse(admin)
	u.Path = "/" + name
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool, u.String()
}

func moduleSchema(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	query, err := os.ReadFile("schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	var out string
	if err := pool.QueryRow(context.Background(), string(query)).Scan(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

// TestGenerateMatchesUpstream replays the model versions with the Go
// generator: same migration statements, same snapshot files and, applied
// to a database (MEDUSA_TEST_PG_ADMIN_URL), the same schema.
func TestGenerateMatchesUpstream(t *testing.T) {
	pool, _ := setupDB(t)
	if pool == nil {
		t.Log("MEDUSA_TEST_PG_ADMIN_URL is not set: the schemas are not compared")
	}
	dir := filepath.Join(t.TempDir(), "migrations")
	base := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	var applied []migrations.Migration
	for i, v := range versions {
		t.Run(v.name, func(t *testing.T) {
			g := loadGolden(t, v.name)
			now := base.Add(time.Duration(i) * time.Second)
			res, err := migrations.Generate(migrations.GenerateOptions{
				SnapshotName: ".snapshot-brand",
				Path:         dir,
				Models:       v.models(),
				Now:          func() time.Time { return now },
			})
			if err != nil {
				t.Fatal(err)
			}
			if g.Up == nil {
				if res.FileName != "" {
					t.Fatalf("upstream detected no changes, Go generated %s", res.FileName)
				}
			} else {
				if res.FileName != "Migration"+now.Format("20060102150405")+".go" {
					t.Fatalf("unexpected migration file %q", res.FileName)
				}
				name, up, down := parseMigration(t, filepath.Join(dir, res.FileName))
				compareStatements(t, "up", g.Up, up)
				compareStatements(t, "down", g.Down, down)
				applied = append(applied, migrations.Migration{Name: name, Up: withoutBlanks(up), Down: withoutBlanks(down)})
			}
			snapshot, err := os.ReadFile(filepath.Join(dir, ".snapshot-brand.json"))
			if err != nil {
				t.Fatal(err)
			}
			if string(snapshot) != g.Snapshot {
				t.Errorf("snapshot differs:\n%s", firstDiff(g.Snapshot, string(snapshot)))
			}
			if pool == nil {
				return
			}
			if _, err := migrations.Run(context.Background(), pool, "brand", applied); err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(filepath.Join("golden", v.name+".schema.txt"))
			if err != nil {
				t.Fatal(err)
			}
			if got := moduleSchema(t, pool); got != strings.TrimSuffix(string(want), "\n") {
				t.Errorf("schema differs:\n%s", firstDiff(strings.TrimSuffix(string(want), "\n"), got))
			}
		})
	}
	// index.go lists the generated migrations
	index, err := os.ReadFile(filepath.Join(dir, "index.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range applied {
		if !strings.Contains(string(index), "\t"+m.Name+",\n") {
			t.Errorf("index.go does not list %s:\n%s", m.Name, index)
		}
	}
}

// TestGenerateCommand runs `medusa db:generate` (the Go command) for a
// project module: the migration, index.go and the snapshot are written into
// the migrations directory of the module, the next run detects no changes
// and unknown modules are reported.
func TestGenerateCommand(t *testing.T) {
	pool, dbURL := setupDB(t)
	if pool == nil {
		t.Skip("MEDUSA_TEST_PG_ADMIN_URL is not set (e.g. postgres://postgres:postgres@localhost:54329/postgres)")
	}
	prev := migrationscripts.SetUniqueTimestampDelay(0)
	defer migrationscripts.SetUniqueTimestampDelay(prev)

	project := t.TempDir()
	module := &typesmodulessdk.ModuleExports{Models: v1()}
	cfg := config.DefineConfig(config.InputConfig{
		ProjectConfig: config.ProjectConfig{DatabaseURL: dbURL},
		Modules:       []config.ModuleConfig{{Key: "brand", Resolve: "./src/modules/brand", Module: module}},
	})
	ctx := context.Background()
	if err := db.Generate(ctx, cfg, db.GenerateOptions{Directory: project, Modules: []string{"brand"}}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(project, "src", "modules", "brand", "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 3 || names[0] != ".snapshot-brand.json" || !strings.HasPrefix(names[1], "Migration") || names[2] != "index.go" {
		t.Fatalf("unexpected migrations directory: %v", names)
	}
	g := loadGolden(t, "v1")
	_, up, down := parseMigration(t, filepath.Join(dir, names[1]))
	compareStatements(t, "up", g.Up, up)
	compareStatements(t, "down", g.Down, down)
	snapshot, _ := os.ReadFile(filepath.Join(dir, ".snapshot-brand.json"))
	if string(snapshot) != g.Snapshot {
		t.Errorf("snapshot differs:\n%s", firstDiff(g.Snapshot, string(snapshot)))
	}

	// no changes: nothing new is written
	if err := db.Generate(ctx, cfg, db.GenerateOptions{Directory: project, Modules: []string{"brand"}}); err != nil {
		t.Fatal(err)
	}
	if again, _ := os.ReadDir(dir); len(again) != 3 {
		t.Fatalf("a migration was generated without changes: %d files", len(again))
	}

	if err := db.Generate(ctx, cfg, db.GenerateOptions{Directory: project, Modules: []string{"unknown"}}); err == nil {
		t.Fatal("expected an error for an unknown module")
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
			from := i - 3
			if from < 0 {
				from = 0
			}
			end := i
			if end > len(wl) {
				end = len(wl)
			}
			return "line " + strconv.Itoa(i+1) + "\ncontext:\n" + strings.Join(wl[from:end], "\n") + "\nwant: " + w + "\ngot:  " + g
		}
	}
	return ""
}
