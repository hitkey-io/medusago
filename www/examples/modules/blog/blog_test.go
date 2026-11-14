package blog

import (
	"context"
	"os"
	"strings"
	"testing"

	typesmodulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
	"github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
)

// The committed migrations must match the models: db:generate would find no
// change.
func TestSnapshotMatchesModels(t *testing.T) {
	snapshot, err := os.ReadFile("migrations/.snapshot-blog.json")
	if err != nil {
		t.Fatal(err)
	}
	up, _, _, err := migrations.SchemaDiff(snapshot, Models)
	if err != nil {
		t.Fatal(err)
	}
	if len(up) != 0 {
		t.Fatalf("the snapshot is out of date: %v", up)
	}
}

func TestModuleService(t *testing.T) {
	svc, err := Module.Service(typesmodulessdk.ServiceOptions{Options: map[string]any{"maxTitleLength": 5}})
	if err != nil {
		t.Fatal(err)
	}
	blogService := svc.(*BlogModuleService)
	if _, err := blogService.CreateDraft(context.Background(), "A long title", "a-long-title"); err == nil || !strings.Contains(err.Error(), "longer than 5") {
		t.Fatalf("unexpected error %v", err)
	}
	if _, err := Module.Service(typesmodulessdk.ServiceOptions{Options: map[string]any{"maxTitleLength": "ten"}}); err == nil {
		t.Fatal("expected an options error")
	}
	for _, name := range []string{"Post", "Comment", "Tag"} {
		if blogService.Internal(name) == nil {
			t.Fatalf("no internal service for %s", name)
		}
	}
}

// The illustration models of the documentation must be valid models.
func TestPropertyModels(t *testing.T) {
	models := []*dml.Entity{Showcase, Article, Product, PostStat, Reference, Page}
	if err := dml.ResolveMeta(models); err != nil {
		t.Fatal(err)
	}
	if _, err := migrations.Generate(migrations.GenerateOptions{SnapshotName: ".snapshot-x", Path: t.TempDir(), Models: models}); err != nil {
		t.Fatal(err)
	}
}

func TestRelationshipModels(t *testing.T) {
	for name, models := range map[string][]*dml.Entity{
		"one-to-one":    {Author, AuthorProfile},
		"optional":      {Writer, WriterProfile},
		"one-sided has": {User, Email},
		"one-sided to":  {Holder, Badge},
		"pivot entity":  {Order, ProductTest, OrdersProducts},
		"searchable":    {Category, Entry},
	} {
		res, err := migrations.Generate(migrations.GenerateOptions{SnapshotName: ".snapshot-x", Path: t.TempDir(), Models: models})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		t.Logf("%s:\n%s", name, strings.Join(res.Up, "\n"))
	}
}

func TestIndexModels(t *testing.T) {
	for name, models := range map[string][]*dml.Entity{"indexes": {Listing}, "checks": {Discount}} {
		res, err := migrations.Generate(migrations.GenerateOptions{SnapshotName: ".snapshot-x", Path: t.TempDir(), Models: models})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		t.Logf("%s:\n%s", name, strings.Join(res.Up, "\n"))
	}
}

func TestPropertySQL(t *testing.T) {
	res, err := migrations.Generate(migrations.GenerateOptions{SnapshotName: ".snapshot-x", Path: t.TempDir(), Models: []*dml.Entity{Showcase, Article}})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(strings.Join(res.Up, "\n"))
}
