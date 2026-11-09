// Package modulesdiff runs module service calls against the Go
// implementation and compares their results with the golden results
// produced by upstream Medusa (see generate.sh).
package modulesdiff

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	modulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/commands/db"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
)

type testCase struct {
	Name   string          `json:"name"`
	Module string          `json:"module"`
	Method string          `json:"method"`
	Args   json.RawMessage `json:"args"`
}

// setupDatabase creates a database migrated by the Go implementation and
// loads seed.sql into it.
func setupDatabase(t *testing.T, name string) *pgxpool.Pool {
	t.Helper()
	admin := os.Getenv("MEDUSA_TEST_PG_ADMIN_URL")
	if admin == "" {
		t.Skip("MEDUSA_TEST_PG_ADMIN_URL is not set (e.g. postgres://postgres:postgres@localhost:54329/postgres)")
	}
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{"drop database if exists " + name, "create database " + name} {
		if _, err := conn.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	conn.Close(ctx)

	u, _ := url.Parse(admin)
	u.Path = "/" + name
	dbURL := u.String()

	cfg := config.DefineConfig(config.InputConfig{ProjectConfig: config.ProjectConfig{DatabaseURL: dbURL}})
	if err := db.Migrate(ctx, cfg, db.MigrateOptions{}); err != nil {
		t.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	seed, err := os.ReadFile("seed.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(seed), pgx.QueryExecModeSimpleProtocol); err != nil {
		t.Fatal(fmt.Errorf("loading seed.sql: %w", err))
	}
	return pool
}

func moduleService(pool *pgxpool.Pool, key string) (*modulessdk.MedusaService, error) {
	exp, ok := modules.Registry[modulessdk.MODULE_PACKAGE_NAMES[key]]
	if !ok {
		return nil, fmt.Errorf("unknown module %s", key)
	}
	return modulessdk.NewMedusaService(pool, exp.Models), nil
}

// run executes one case and returns it encoded like run-calls.js does.
func run(ctx context.Context, svc *modulessdk.MedusaService, c testCase) ([]byte, error) {
	decoded, err := dal.DecodeOrderedJSON(c.Args)
	if err != nil {
		return nil, err
	}
	args, _ := decoded.([]any)
	result, err := svc.Call(ctx, c.Method, args)
	if err != nil {
		if me, ok := common.AsMedusaError(err); ok {
			e := dal.NewRecord()
			e.Set("type", me.Type)
			e.Set("message", me.Message)
			out := dal.NewRecord()
			out.Set("error", e)
			return out.MarshalJSON()
		}
		return nil, err
	}
	encoded, err := encodeResult(result)
	if err != nil {
		return nil, err
	}
	return []byte(`{"result":` + string(encoded) + `}`), nil
}

func encodeResult(v any) ([]byte, error) {
	switch t := v.(type) {
	case []any:
		parts := make([]string, len(t))
		for i, item := range t {
			b, err := encodeResult(item)
			if err != nil {
				return nil, err
			}
			parts[i] = string(b)
		}
		return []byte("[" + strings.Join(parts, ",") + "]"), nil
	default:
		return dal.MarshalValue(v)
	}
}

// firstDifference returns a readable description of the first difference.
func firstDifference(path string, want, got any) string {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return fmt.Sprintf("%s: want object, got %v", path, got)
		}
		keys := map[string]bool{}
		for k := range w {
			keys[k] = true
		}
		for k := range g {
			keys[k] = true
		}
		sorted := make([]string, 0, len(keys))
		for k := range keys {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)
		for _, k := range sorted {
			wv, wok := w[k]
			gv, gok := g[k]
			if !wok {
				return fmt.Sprintf("%s.%s: unexpected key (got %v)", path, k, gv)
			}
			if !gok {
				return fmt.Sprintf("%s.%s: missing key (want %v)", path, k, wv)
			}
			if d := firstDifference(path+"."+k, wv, gv); d != "" {
				return d
			}
		}
	case []any:
		g, ok := got.([]any)
		if !ok {
			return fmt.Sprintf("%s: want array, got %v", path, got)
		}
		if len(w) != len(g) {
			return fmt.Sprintf("%s: want %d items, got %d", path, len(w), len(g))
		}
		for i := range w {
			if d := firstDifference(fmt.Sprintf("%s[%d]", path, i), w[i], g[i]); d != "" {
				return d
			}
		}
	default:
		if !reflect.DeepEqual(want, got) {
			return fmt.Sprintf("%s: want %v, got %v", path, want, got)
		}
	}
	return ""
}

// orderedKeys lists every object's keys in order, to compare JSON key order.
func orderedKeys(data []byte) (string, error) {
	v, err := dal.DecodeOrderedJSON(data)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	var walk func(any)
	walk = func(v any) {
		switch t := v.(type) {
		case *dal.Record:
			b.WriteString("{" + strings.Join(t.Keys(), ",") + "}")
			for _, k := range t.Keys() {
				x, _ := t.Get(k)
				walk(x)
			}
		case []any:
			for _, x := range t {
				walk(x)
			}
		}
	}
	walk(v)
	return b.String(), nil
}

func TestModulesMatchUpstream(t *testing.T) {
	files, err := filepath.Glob("cases/*.json")
	if err != nil {
		t.Fatal(err)
	}
	pool := setupDatabase(t, "medusa_diff_go")
	ctx := context.Background()

	for _, file := range files {
		var cases []testCase
		raw, _ := os.ReadFile(file)
		if err := json.Unmarshal(raw, &cases); err != nil {
			t.Fatal(err)
		}
		goldenRaw, err := os.ReadFile(filepath.Join("golden", filepath.Base(file)))
		if err != nil {
			t.Fatal(err)
		}
		golden := map[string]json.RawMessage{}
		if err := json.Unmarshal(goldenRaw, &golden); err != nil {
			t.Fatal(err)
		}

		for _, c := range cases {
			t.Run(c.Name, func(t *testing.T) {
				svc, err := moduleService(pool, c.Module)
				if err != nil {
					t.Fatal(err)
				}
				got, err := run(ctx, svc, c)
				if err != nil {
					t.Fatal(err)
				}
				var wantV, gotV any
				_ = json.Unmarshal(golden[c.Name], &wantV)
				if err := json.Unmarshal(got, &gotV); err != nil {
					t.Fatalf("invalid output: %v\n%s", err, got)
				}
				if d := firstDifference("$", wantV, gotV); d != "" {
					t.Fatalf("%s\n got: %.600s", d, got)
				}
				wantKeys, _ := orderedKeys(golden[c.Name])
				gotKeys, _ := orderedKeys(got)
				if wantKeys != gotKeys {
					t.Fatalf("key order differs:\nwant %.400s\n got %.400s", wantKeys, gotKeys)
				}
			})
		}
	}
}
