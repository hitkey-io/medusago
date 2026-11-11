package common

import (
	"encoding/json"
	"os"
	"testing"
)

// The vectors are the outputs of the upstream helpers (pluralize,
// singularize, toPascalCase, toCamelCase, kebabCase, camelToSnakeCase,
// upperCaseFirst, lowerCaseFirst) from @medusajs/framework/utils.
func TestStringCaseVectors(t *testing.T) {
	data, err := os.ReadFile("testdata/string-case-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []map[string]string
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	fns := map[string]func(string) string{
		"plural":   Pluralize,
		"singular": Singularize,
		"pascal":   ToPascalCase,
		"camel":    ToCamelCase,
		"kebab":    KebabCase,
		"snake":    CamelToSnakeCase,
		"upper":    UpperCaseFirst,
		"lower":    LowerCaseFirst,
	}
	for _, v := range vectors {
		for name, fn := range fns {
			if got := fn(v["w"]); got != v[name] {
				t.Errorf("%s(%q) = %q, want %q", name, v["w"], got, v[name])
			}
		}
	}
}
