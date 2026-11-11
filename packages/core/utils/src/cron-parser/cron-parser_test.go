package cronparser

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// TestVectors compares two next() of each expression with cron-parser
// (testdata/vectors.json, generated with parseExpression(e, { currentDate,
// tz: "UTC" })).
func TestVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors [][4]string
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	for _, v := range vectors {
		cur, _ := time.Parse(time.RFC3339Nano, v[1])
		e, err := Parse(v[0], Options{CurrentDate: cur, TZ: "UTC"})
		if err != nil {
			t.Fatalf("%s: %v", v[0], err)
		}
		for i := 2; i < 4; i++ {
			n, err := e.Next()
			if err != nil {
				t.Fatalf("%s: %v", v[0], err)
			}
			if got := n.UTC().Format("2006-01-02T15:04:05.000Z"); got != v[i] {
				t.Errorf("%s from %s: %s, want %s", v[0], v[1], got, v[i])
			}
		}
	}
}
