package middlewares

import (
	"encoding/json"
	"os"
	"testing"
)

func TestV8JSONErrors(t *testing.T) {
	raw, err := os.ReadFile("testdata/v8-json-errors.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases [][2]*string
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		got := ValidateJSONLikeV8(*c[0])
		switch {
		case c[1] == nil && got != nil:
			t.Errorf("%q: unexpected error %v", *c[0], got)
		case c[1] != nil && (got == nil || got.Error() != *c[1]):
			t.Errorf("%q: got %v want %s", *c[0], got, *c[1])
		}
	}
}
