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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	modulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/commands/db"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
)

type testCase struct {
	Name   string          `json:"name"`
	Module string          `json:"module"`
	Method string          `json:"method"`
	Args   json.RawMessage `json:"args"`
	// Service calls the module's own service (its overridden methods)
	// instead of the generated MedusaService.
	Service bool `json:"service"`
	// SortBy maps dotted output paths ("*" for every array item) of arrays
	// upstream returns in a nondeterministic order (e.g. rows of a hash join)
	// to the item field they are compared sorted by.
	SortBy map[string]string `json:"sortby"`
}

// setupDatabase creates a database migrated by the Go implementation and
// loads seed.sql into it.
func setupDatabase(t *testing.T, name string) *pgxpool.Pool {
	t.Helper()
	// MEDUSA_TEST_DB_PREFIX lets parallel runs (other checkouts) use their own databases.
	name = os.Getenv("MEDUSA_TEST_DB_PREFIX") + name
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

	// Same as generate.sh: keep the schema and the migration tracking
	// tables, replace the data with seed.sql.
	if _, err := pool.Exec(ctx, `
    DO $$ DECLARE t text; BEGIN
      FOR t IN SELECT tablename FROM pg_tables WHERE schemaname = 'public'
        AND tablename NOT IN ('mikro_orm_migrations', 'script_migrations', 'link_module_migrations')
      LOOP EXECUTE format('TRUNCATE %I CASCADE', t); END LOOP;
    END $$;`); err != nil {
		t.Fatal(err)
	}
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
	return modulessdk.NewMedusaService(pool, exp.Models, exp.JoinerConfig), nil
}

var refRe = regexp.MustCompile(`^\{\{([^.}]+)\.([^}]+)\}\}$`)

// resolveRefs mirrors resolveRefs from run-calls.js.
func resolveRefs(v any, outputs map[string]any) any {
	switch t := v.(type) {
	case string:
		m := refRe.FindStringSubmatch(t)
		if m == nil {
			return t
		}
		var cur any = outputs[m[1]]
		for _, key := range strings.Split(m[2], ".") {
			switch c := cur.(type) {
			case map[string]any:
				cur = c[key]
			case []any:
				i, err := strconv.Atoi(key)
				if err != nil || i >= len(c) {
					return nil
				}
				cur = c[i]
			default:
				return nil
			}
		}
		return cur
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = resolveRefs(x, outputs)
		}
		return out
	case *dal.Record:
		out := dal.NewRecord()
		for _, k := range t.Keys() {
			x, _ := t.Get(k)
			out.Set(k, resolveRefs(x, outputs))
		}
		return out
	default:
		return v
	}
}

var (
	generatedIDRe = regexp.MustCompile(`^([a-z]+_)*[0-9A-HJKMNP-TV-Z]{26}$`)
	embeddedIDRe  = regexp.MustCompile(`\b(?:[a-z]+_)+[0-9A-HJKMNP-TV-Z]{26}\b`)
	timestampRe   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$`)
)

// knownValues collects the identifiers and timestamps present in seed.sql.
func knownValues(seed string) map[string]bool {
	known := map[string]bool{}
	for _, m := range regexp.MustCompile(`'([^']*)'`).FindAllStringSubmatch(seed, -1) {
		known[m[1]] = true
		if t, err := time.Parse("2006-01-02 15:04:05.999999-07", m[1]); err == nil {
			known[dal.FormatJSDate(t)] = true
		}
	}
	return known
}

// normalize replaces identifiers and timestamps generated while the cases
// run (absent from seed.sql) with stable placeholders.
func normalize(v any, known map[string]bool, ids map[string]string) any {
	switch t := v.(type) {
	case string:
		if known[t] {
			return t
		}
		if timestampRe.MatchString(t) {
			return "<now>"
		}
		if generatedIDRe.MatchString(t) {
			if p, ok := ids[t]; ok {
				return p
			}
			ids[t] = fmt.Sprintf("<generated %d>", len(ids)+1)
			return ids[t]
		}
		// generated identifiers inside messages
		return embeddedIDRe.ReplaceAllStringFunc(t, func(id string) string {
			if known[id] {
				return id
			}
			if p, ok := ids[id]; ok {
				return p
			}
			ids[id] = fmt.Sprintf("<generated %d>", len(ids)+1)
			return ids[id]
		})
	case []any:
		for i := range t {
			t[i] = normalize(t[i], known, ids)
		}
		return t
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			t[k] = normalize(t[k], known, ids)
		}
		return t
	default:
		return v
	}
}

// graphInput converts the query.graph argument.
func graphInput(arg any) query.GraphInput {
	in := query.GraphInput{}
	r, _ := arg.(*dal.Record)
	if r == nil {
		return in
	}
	if v, ok := r.Get("entity"); ok {
		in.Entity = fmt.Sprint(v)
	}
	if v, ok := r.Get("fields"); ok {
		for _, f := range v.([]any) {
			in.Fields = append(in.Fields, fmt.Sprint(f))
		}
	}
	if v, ok := r.Get("filters"); ok {
		in.Filters, _ = v.(*dal.Record)
	}
	if v, ok := r.Get("pagination"); ok {
		in.Pagination, _ = v.(*dal.Record)
	}
	if v, ok := r.Get("withDeleted"); ok {
		in.WithDeleted, _ = v.(bool)
	}
	return in
}

// run executes one case and returns it encoded like run-calls.js does.
func run(ctx context.Context, svc *modulessdk.MedusaService, custom any, q *query.Query, c testCase, outputs map[string]any) ([]byte, error) {
	decoded, err := dal.DecodeOrderedJSON(c.Args)
	if err != nil {
		return nil, err
	}
	args, _ := resolveRefs(decoded, outputs).([]any)
	var result any
	if c.Module == "query" {
		var arg any
		if len(args) > 0 {
			arg = args[0]
		}
		result, err = q.Graph(ctx, graphInput(arg), joiner.Options{})
	} else if custom != nil {
		result, err = callService(ctx, custom, c.Method, args)
	} else {
		result, err = svc.Call(ctx, c.Method, args)
	}
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
	case *query.GraphResult:
		return t.MarshalJSON()
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
func orderedKeys(data []byte, unorderedArrays bool) (string, error) {
	v, err := dal.DecodeOrderedJSON(data)
	if err != nil {
		return "", err
	}
	var walk func(any) string
	walk = func(v any) string {
		switch t := v.(type) {
		case *dal.Record:
			s := "{" + strings.Join(t.Keys(), ",") + "}"
			for _, k := range t.Keys() {
				x, _ := t.Get(k)
				s += walk(x)
			}
			return s
		case []any:
			items := make([]string, len(t))
			for i, x := range t {
				items[i] = walk(x)
			}
			if unorderedArrays {
				sort.Strings(items)
			}
			return strings.Join(items, "")
		}
		return ""
	}
	return walk(v), nil
}

func TestModulesMatchUpstream(t *testing.T) {
	files, err := filepath.Glob("cases/*.json")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	seed, err := os.ReadFile("seed.sql")
	if err != nil {
		t.Fatal(err)
	}
	known := knownValues(string(seed))

	for _, file := range files {
		pool := setupDatabase(t, "medusa_diff_go_"+strings.TrimSuffix(strings.ReplaceAll(filepath.Base(file), "-", "_"), ".json"))
		outputs := map[string]any{}
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
				var svc *modulessdk.MedusaService
				if c.Module != "query" && !c.Service {
					svc, err = moduleService(pool, c.Module)
					if err != nil {
						t.Fatal(err)
					}
				}
				var custom any
				if c.Service {
					if custom, err = moduleServiceInstance(pool, c.Module); err != nil {
						t.Fatal(err)
					}
				}
				q, err := loaders.NewDefaultQuery(pool)
				if err != nil {
					t.Fatal(err)
				}
				got, err := run(ctx, svc, custom, q, c, outputs)
				if err != nil {
					t.Fatal(err)
				}
				var wantV, gotV any
				_ = json.Unmarshal(golden[c.Name], &wantV)
				if err := json.Unmarshal(got, &gotV); err != nil {
					t.Fatalf("invalid output: %v\n%s", err, got)
				}
				var saved any
				_ = json.Unmarshal(got, &saved)
				outputs[c.Name] = saved
				for path, field := range c.SortBy {
					wantV = sortAt(wantV, strings.Split(path, "."), field)
					gotV = sortAt(gotV, strings.Split(path, "."), field)
				}
				wantV = normalize(wantV, known, map[string]string{})
				gotV = normalize(gotV, known, map[string]string{})
				if d := firstDifference("$", wantV, gotV); d != "" {
					t.Fatalf("%s\n got: %.600s", d, got)
				}
				wantKeys, _ := orderedKeys(golden[c.Name], len(c.SortBy) > 0)
				gotKeys, _ := orderedKeys(got, len(c.SortBy) > 0)
				if wantKeys != gotKeys {
					t.Fatalf("key order differs:\nwant %.400s\n got %.400s", wantKeys, gotKeys)
				}
			})
		}
	}
}
