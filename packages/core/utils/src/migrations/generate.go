package migrations

import (
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal/mikro-orm/schema"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
)

// This file mirrors Migrations.generate (migrations/index.ts) over the
// MikroORM migrator (Migrator.createMigration with a snapshot) and the
// CustomTsMigrationGenerator of mikro-orm-create-connection.ts. The
// migration is written as Go code (a Migration variable in package
// migrations) instead of a TypeScript class, and index.go lists the
// migrations of the directory.

// GoImportPath is the import path of this package in generated files.
const GoImportPath = "github.com/hitkey-io/medusago/packages/core/utils/src/migrations"

// GenerateOptions are the inputs of Generate.
type GenerateOptions struct {
	// SnapshotName is the snapshot file name without extension
	// (".snapshot-<module>").
	SnapshotName string
	// Path is the migrations directory.
	Path string
	// Models are the module models.
	Models []*dml.Entity
	// Now returns the time the migration is generated at (time.Now).
	Now func() time.Time
}

// GenerateResult mirrors MigrationResult: no FileName when nothing changed.
type GenerateResult struct {
	FileName string
	Code     string
	// Up and Down are the statements of the migration, "" separating the
	// groups (the diff the generator receives, before its patches).
	Up   []string
	Down []string
}

// Generate mirrors Migrations.generate.
func Generate(opts GenerateOptions) (*GenerateResult, error) {
	snapshotPath := filepath.Join(opts.Path, opts.SnapshotName+".json")
	if err := migrateSnapshotFile(snapshotPath); err != nil {
		return nil, err
	}
	if err := ensureSnapshot(snapshotPath); err != nil {
		return nil, err
	}
	return createMigration(opts, snapshotPath)
}

// migrateSnapshotFile mirrors Migrations.migrateSnapshotFile.
func migrateSnapshotFile(snapshotPath string) error {
	if _, err := os.Stat(snapshotPath); err == nil {
		return nil
	}
	expected := filepath.Base(snapshotPath)
	legacy := expected
	if strings.HasPrefix(expected, ".snapshot-medusa-") {
		legacy = ".snapshot-" + strings.TrimPrefix(expected, ".snapshot-medusa-")
	}
	entries, err := os.ReadDir(filepath.Dir(snapshotPath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.Type().IsRegular() && e.Name() == legacy {
			abs := filepath.Join(filepath.Dir(snapshotPath), e.Name())
			if abs != snapshotPath {
				return os.Rename(abs, snapshotPath)
			}
		}
	}
	return nil
}

// ensureSnapshot mirrors Migrations.ensureSnapshot.
func ensureSnapshot(snapshotPath string) error {
	if err := os.MkdirAll(filepath.Dir(snapshotPath), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(snapshotPath); err == nil {
		return nil
	}
	return os.WriteFile(snapshotPath, []byte(schema.EmptySnapshot), 0o644)
}

// createMigration mirrors Migrator.createMigration (not initial, not
// blank).
func createMigration(opts GenerateOptions, snapshotPath string) (*GenerateResult, error) {
	if err := os.MkdirAll(opts.Path, 0o755); err != nil {
		return nil, err
	}
	metadata, err := dml.SchemaMetadata(opts.Models)
	if err != nil {
		return nil, err
	}
	up, down, err := schemaDiff(snapshotPath, metadata)
	if err != nil {
		return nil, err
	}
	if len(up) == 0 {
		return &GenerateResult{Up: up, Down: down}, nil
	}
	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	timestamp := now().UTC().Format("20060102150405")
	className := "Migration" + timestamp
	fileName := className + ".go"
	upStmts, downStmts := migrationStatements(up, down)
	code, err := migrationFile(className, upStmts, downStmts)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(opts.Path, fileName), code, 0o644); err != nil {
		return nil, err
	}
	if err := writeIndex(opts.Path); err != nil {
		return nil, err
	}
	// storeCurrentSchema
	if err := os.WriteFile(snapshotPath, []byte(schema.TargetSchema(metadata).JSON()), 0o644); err != nil {
		return nil, err
	}
	return &GenerateResult{FileName: fileName, Code: string(code), Up: up, Down: down}, nil
}

// schemaDiff mirrors Migrator.getSchemaDiff: the update migration from the
// snapshot to the models, split into statements.
func schemaDiff(snapshotPath string, metadata []*dml.SchemaEntity) ([]string, []string, error) {
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		return nil, nil, err
	}
	from, err := schema.FromSnapshot(data)
	if err != nil {
		return nil, nil, err
	}
	diff := schema.UpdateSchemaMigrationSQL(from, metadata)
	return cleanUp(splitStatements(diff.Up)), cleanUp(splitStatements(diff.Down)), nil
}

// SchemaDiff returns the statements Generate would write for the given
// snapshot content and models (for tests).
func SchemaDiff(snapshot []byte, models []*dml.Entity) (up, down []string, snapshotJSON string, err error) {
	metadata, err := dml.SchemaMetadata(models)
	if err != nil {
		return nil, nil, "", err
	}
	from, err := schema.FromSnapshot(snapshot)
	if err != nil {
		return nil, nil, "", err
	}
	diff := schema.UpdateSchemaMigrationSQL(from, metadata)
	up, down = cleanUp(splitStatements(diff.Up)), cleanUp(splitStatements(diff.Down))
	up, down = migrationStatements(up, down)
	return up, down, schema.TargetSchema(metadata).JSON(), nil
}

// splitStatements mirrors the splitStatements of Migrator.getSchemaDiff.
func splitStatements(sql string) []string {
	var result []string
	buf := ""
	for _, chunk := range strings.Split(sql, ";\n") {
		if buf != "" {
			buf += ";\n"
		}
		buf += chunk
		if len(strings.Split(buf, "'"))%2 == 0 {
			continue
		}
		if strings.HasPrefix(buf, "\n") {
			result = append(result, "")
		}
		trimmed := strings.TrimSpace(buf)
		if trimmed != "" {
			if !strings.HasSuffix(trimmed, ";") {
				trimmed += ";"
			}
			result = append(result, trimmed)
		}
		buf = ""
	}
	return result
}

func cleanUp(diff []string) []string {
	for len(diff) > 0 && diff[len(diff)-1] == "" {
		diff = diff[:len(diff)-1]
	}
	return diff
}

var (
	uniqueIndexNameRe = regexp.MustCompile(`"IDX_(.+?)_unique"`)
	onTableRe         = regexp.MustCompile(`ON "(.+?)"`)
)

// migrationStatements mirrors CustomTsMigrationGenerator.generateMigrationFile
// and createStatement: the unique constraints named like a DML unique index
// are dropped before the up statements, and every statement goes through
// normalizeMigrationSQL. "" entries are the blank lines between groups.
func migrationStatements(up, down []string) ([]string, []string) {
	var patches []string
	for _, sql := range up {
		m := uniqueIndexNameRe.FindStringSubmatch(sql)
		if m == nil {
			continue
		}
		if t := onTableRe.FindStringSubmatch(sql); t != nil {
			patches = append(patches, fmt.Sprintf(`alter table if exists "%s" drop constraint if exists "%s_unique";`, t[1], m[1]))
		}
	}
	var outUp []string
	for i := len(patches) - 1; i >= 0; i-- {
		outUp = append(outUp, patches[i])
	}
	for _, sql := range up {
		outUp = append(outUp, NormalizeMigrationSQL(sql))
	}
	var outDown []string
	for _, sql := range down {
		outDown = append(outDown, NormalizeMigrationSQL(sql))
	}
	return outUp, outDown
}

var normalizations = []struct{ from, to string }{
	{"create table ", "create table if not exists "},
	{"alter table ", "alter table if exists "},
	{"create index ", "create index if not exists "},
	{"alter index ", "alter index if exists "},
	{"drop index ", "drop index if exists "},
	{"create unique index ", "create unique index if not exists "},
	{"drop unique index ", "drop unique index if exists "},
	{"add column ", "add column if not exists "},
	{"drop column ", "drop column if exists "},
	{"drop constraint ", "drop constraint if exists "},
}

// NormalizeMigrationSQL mirrors normalizeMigrationSQL (dal/utils.ts): the
// statements get "if (not) exists" clauses.
func NormalizeMigrationSQL(sql string) string {
	for _, n := range normalizations {
		guard := strings.TrimPrefix(n.to, n.from)
		var b strings.Builder
		rest := sql
		for {
			i := strings.Index(rest, n.from)
			if i < 0 {
				b.WriteString(rest)
				break
			}
			b.WriteString(rest[:i+len(n.from)])
			rest = rest[i+len(n.from):]
			if !strings.HasPrefix(rest, guard) {
				b.WriteString(guard)
			}
		}
		sql = b.String()
	}
	return sql
}

// goString returns a Go literal of s.
func goString(s string) string {
	if !strings.Contains(s, "`") && !strings.Contains(s, "\r") {
		return "`" + s + "`"
	}
	return strconv.Quote(s)
}

func statementsLiteral(b *strings.Builder, field string, stmts []string) {
	fmt.Fprintf(b, "\t%s: []string{\n", field)
	for _, s := range stmts {
		if s == "" {
			b.WriteString("\n")
			continue
		}
		fmt.Fprintf(b, "\t\t%s,\n", goString(s))
	}
	b.WriteString("\t},\n")
}

// migrationFile is the Go version of the TSMigrationGenerator output.
func migrationFile(className string, up, down []string) ([]byte, error) {
	var b strings.Builder
	b.WriteString("package migrations\n\n")
	fmt.Fprintf(&b, "import (\n\tm %q\n)\n\n", GoImportPath)
	fmt.Fprintf(&b, "// %s was generated by `medusa db:generate`.\n", className)
	fmt.Fprintf(&b, "var %s = m.Migration{\n", className)
	fmt.Fprintf(&b, "\tName: %q,\n", className)
	statementsLiteral(&b, "Up", up)
	if len(down) > 0 {
		statementsLiteral(&b, "Down", down)
	}
	b.WriteString("}\n")
	return format.Source([]byte(b.String()))
}

var migrationVarRe = regexp.MustCompile(`(?m)^var (Migration\w+) = m\.Migration\{`)

// writeIndex writes index.go: the Migrations list of the directory, by
// name like the MikroORM migrator runs them.
func writeIndex(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if !e.Type().IsRegular() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") || e.Name() == "index.go" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		for _, m := range migrationVarRe.FindAllStringSubmatch(string(data), -1) {
			names = append(names, m[1])
		}
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString("// Code generated by medusa db:generate. DO NOT EDIT.\n\n")
	b.WriteString("// Package migrations holds the database migrations of the module.\n")
	b.WriteString("package migrations\n\n")
	fmt.Fprintf(&b, "import (\n\tm %q\n)\n\n", GoImportPath)
	b.WriteString("// Migrations lists the module migrations in execution order.\n")
	b.WriteString("var Migrations = []m.Migration{\n")
	for _, n := range names {
		fmt.Fprintf(&b, "\t%s,\n", n)
	}
	b.WriteString("}\n")
	src, err := format.Source([]byte(b.String()))
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "index.go"), src, 0o644)
}
