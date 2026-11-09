package modules

import (
	"context"
	"os"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
)

// persistedColumns returns the columns an entity stores in its own table.
func persistedColumns(e *dml.Entity) []string {
	var cols []string
	for _, p := range e.Meta.Properties {
		if p.Persist != nil && !*p.Persist {
			continue
		}
		switch p.Kind {
		case "scalar", "m:1":
			cols = append(cols, p.FieldNames...)
		case "1:1":
			if p.Owner {
				cols = append(cols, p.FieldNames...)
			}
		}
	}
	sort.Strings(cols)
	return slices.Compact(cols)
}

// legacyColumns are columns created by upstream migrations that are no longer
// part of the upstream models; they exist in every Medusa database.
var legacyColumns = map[string][]string{
	// Created by order Migration20240219102530, dropped from the model later.
	"order_line_item": {"totals_id"},
}

func allModels() map[string][]*dml.Entity {
	out := map[string][]*dml.Entity{}
	for name, exp := range Registry {
		if len(exp.Models) > 0 {
			out[name] = exp.Models
		}
	}
	return out
}

func TestModelsAreConsistent(t *testing.T) {
	count := 0
	for pkg, models := range allModels() {
		for _, e := range models {
			count++
			if e.Meta.Collection == "" || len(e.Meta.PrimaryKey) == 0 {
				t.Errorf("%s %s: missing table or primary key", pkg, e.Name)
			}
			for _, p := range e.Properties {
				if p.Relation == nil {
					continue
				}
				if _, ok := e.MetaProperty(p.Name); !ok {
					t.Errorf("%s %s: relation %s has no resolved metadata", pkg, e.Name, p.Name)
				}
			}
		}
	}
	if count == 0 {
		t.Fatal("no models registered")
	}
}

// TestModelsMatchDatabase compares the columns of every model with the table
// created by the migrations. It needs a database migrated by `medusa
// db:migrate` (with every module enabled whose models are checked) in
// MEDUSA_TEST_DATABASE_URL.
func TestModelsMatchDatabase(t *testing.T) {
	url := os.Getenv("MEDUSA_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("MEDUSA_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	checked := 0
	for pkg, models := range allModels() {
		for _, e := range models {
			rows, err := db.Query(ctx,
				`SELECT column_name FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1 ORDER BY column_name`,
				e.Meta.Collection)
			if err != nil {
				t.Fatal(err)
			}
			actual, err := pgx.CollectRows(rows, pgx.RowTo[string])
			if err != nil {
				t.Fatal(err)
			}
			if len(actual) == 0 {
				continue // module not enabled in this database
			}
			checked++
			actual = slices.DeleteFunc(actual, func(c string) bool {
				return slices.Contains(legacyColumns[e.Meta.Collection], c)
			})
			if want := persistedColumns(e); strings.Join(want, ",") != strings.Join(actual, ",") {
				t.Errorf("%s %s (%s):\n model: %v\n table: %v", pkg, e.Name, e.Meta.Collection, want, actual)
			}
		}
	}
	t.Logf("checked %d models against the database", checked)
}
