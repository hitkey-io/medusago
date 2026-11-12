package commands

import (
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveAdminPlugins(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "apps", "backend")
	// installed in a parent node_modules (workspace hoisting)
	writeFile(t, filepath.Join(root, "node_modules", "@medusajs", "draft-order", "package.json"),
		`{"name":"@medusajs/draft-order","exports":{"./admin":{"import":"./a.mjs"}}}`)
	// no admin export
	writeFile(t, filepath.Join(project, "node_modules", "server-only", "package.json"), `{"name":"server-only","exports":{".":"./i.js"}}`)
	// local plugin with srcDir
	writeFile(t, filepath.Join(project, "plugins", "local", "package.json"), `{"name":"local-plugin"}`)
	writeFile(t, filepath.Join(project, "plugins", "local", ".medusa", "server", "medusa-plugin-options.json"), `{"srcDir":"/src/local"}`)

	sources, packages := resolveAdminPlugins(project, []config.PluginConfig{
		{Resolve: "@medusajs/draft-order"},
		{Resolve: "server-only"},
		{Resolve: "./plugins/local"},
		{Resolve: "missing-plugin"},
	}, slog.Default())
	if !reflect.DeepEqual(packages, []string{"@medusajs/draft-order/admin"}) {
		t.Fatalf("packages = %v", packages)
	}
	if !reflect.DeepEqual(sources, []string{filepath.Join("/src/local", "admin")}) {
		t.Fatalf("sources = %v", sources)
	}
}
