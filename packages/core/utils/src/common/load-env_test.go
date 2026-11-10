package common

import (
	"os"
	"path/filepath"
	"testing"
)

const testDotenv = "DB_NAME=medusa-shop\n" +
	"DATABASE_URL=postgres://postgres@localhost/$DB_NAME\n" +
	"export A=\"x\\ny\" # c\n" +
	"B=plain value # comment\n" +
	"C=${MISSING:-def}\n" +
	"D=${DB_NAME}-x\n" +
	"E=\\$NOT\n" +
	"F='single $DB_NAME'\n" +
	"G=`bt`\n" +
	"H=\n" +
	"I=${UNDEF}end\n" +
	"J=$(cmd)\n" +
	"K=a#b\n" +
	"L=\"multi\nline\"\n" +
	"M=$B\n" +
	"O=${DB_NAME:-zz}/x\n"

// Expected values recorded with dotenv 18.0.9 + dotenv-expand 11.0.7:
// expand(dotenv.config({ path: [".env.production", ".env"] })), optionally
// preceded by dotenv.config() like the medusa CLI entry point.
var loadEnvVectors = map[bool]map[string]string{
	false: {
		"A": "x\ny", "B": "prod", "C": "def", "D": "medusa-shop-x",
		"DATABASE_URL": "postgres://postgres@localhost/medusa-shop", "DB_NAME": "medusa-shop",
		"E": "$NOT", "F": "single medusa-shop", "G": "bt", "H": "", "I": "end", "J": "$(cmd)",
		"K": "a", "L": "multi\nline", "M": "prod", "N": "only-prod", "O": "medusa-shop/x",
	},
	true: {
		"A": "x\ny", "B": "plain value", "C": "def", "D": "medusa-shop-x",
		"DATABASE_URL": "postgres://postgres@localhost/medusa-shop", "DB_NAME": "medusa-shop",
		"E": "$NOT", "F": "single medusa-shop", "G": "bt", "H": "", "I": "end", "J": "$(cmd)",
		"K": "a", "L": "multi\nline", "M": "plain value", "N": "only-prod", "O": "medusa-shop/x",
	},
}

func TestLoadEnv(t *testing.T) {
	for _, cliEntry := range []bool{false, true} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(testDotenv), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".env.production"), []byte("B=prod\nN=only-prod\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		want := loadEnvVectors[cliEntry]
		for k := range want {
			// t.Setenv registers the restore; unset for the test itself.
			t.Setenv(k, "")
			os.Unsetenv(k)
		}
		if cliEntry {
			ConfigDotenv(filepath.Join(dir, ".env"))
		}
		LoadEnv("production", dir)
		for k, v := range want {
			if got := os.Getenv(k); got != v {
				t.Errorf("cli=%v %s = %q, want %q", cliEntry, k, got, v)
			}
		}
	}
}
