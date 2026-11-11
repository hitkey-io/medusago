package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const jsonConfig = `{
  "projectConfig": {
    "databaseUrl": "postgres://localhost/medusa",
    "redisUrl": "redis://localhost:6379",
    "redisPrefix": "shop:",
    "redisOptions": { "db": 2, "tls": {} },
    "http": { "storeCors": "http://localhost:8000", "jwtSecret": "secret" },
    "sessionOptions": { "name": "sid", "ttl": 3600000 }
  },
  "admin": { "disable": true },
  "modules": [{ "resolve": "@medusajs/medusa/translation" }]
}`

const yamlConfig = `# medusa-config.yaml
projectConfig:
  databaseUrl: postgres://localhost/medusa
  redisUrl: redis://localhost:6379
  redisPrefix: "shop:"
  redisOptions:
    db: 2
    tls: {}
  http:
    storeCors: http://localhost:8000
    jwtSecret: secret
  sessionOptions:
    name: sid
    ttl: 3600000
admin:
  disable: true
modules:
  - resolve: "@medusajs/medusa/translation"
`

func TestGetConfigFileYAML(t *testing.T) {
	jsonDir, yamlDir, ymlDir := t.TempDir(), t.TempDir(), t.TempDir()
	write := func(dir, name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(jsonDir, "medusa-config.json", jsonConfig)
	write(yamlDir, "medusa-config.yaml", yamlConfig)
	write(ymlDir, "medusa-config.yml", yamlConfig)

	want, path, err := GetConfigFile(jsonDir, "medusa-config")
	if err != nil || path == "" {
		t.Fatalf("json: %v %q", err, path)
	}
	for _, dir := range []string{yamlDir, ymlDir} {
		got, path, err := GetConfigFile(dir, "medusa-config")
		if err != nil || path == "" {
			t.Fatalf("yaml: %v %q", err, path)
		}
		if !reflect.DeepEqual(DefineConfig(want), DefineConfig(got)) {
			t.Fatalf("%s: yaml config differs from json:\n%#v\n%#v", path, want, got)
		}
	}

	// the JSON form wins when both exist
	write(yamlDir, "medusa-config.json", `{"projectConfig": {"databaseUrl": "postgres://json"}}`)
	got, path, err := GetConfigFile(yamlDir, "medusa-config")
	if err != nil || filepath.Base(path) != "medusa-config.json" || got.ProjectConfig.DatabaseURL != "postgres://json" {
		t.Fatalf("lookup order: %v %q %q", err, path, got.ProjectConfig.DatabaseURL)
	}

	// no file: defaults
	if _, path, err := GetConfigFile(t.TempDir(), "medusa-config"); err != nil || path != "" {
		t.Fatalf("missing: %v %q", err, path)
	}

	// invalid YAML reports the file
	bad := t.TempDir()
	write(bad, "medusa-config.yaml", "projectConfig: [")
	if _, path, err := GetConfigFile(bad, "medusa-config"); err == nil || filepath.Base(path) != "medusa-config.yaml" {
		t.Fatalf("invalid yaml: %v %q", err, path)
	}
}
