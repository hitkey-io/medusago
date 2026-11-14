package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasAdminSources(t *testing.T) {
	root := t.TempDir()
	write := func(name string) {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dir := filepath.Join(root, "src", "admin")

	if hasAdminSources(dir) {
		t.Fatal("a missing directory has no sources")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if hasAdminSources(dir) {
		t.Fatal("an empty directory has no sources")
	}
	write("src/admin/.gitkeep")
	write("src/admin/README.md")
	if hasAdminSources(dir) {
		t.Fatal("placeholders are not extension sources")
	}
	write("src/admin/widgets/product-brand.tsx")
	if !hasAdminSources(dir) {
		t.Fatal("a widget is an extension source")
	}
}
