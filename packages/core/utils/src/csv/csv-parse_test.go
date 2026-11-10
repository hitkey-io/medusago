package csv

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// testdata/csv-parse.json holds csv-parse 7.0.3 results (columns: true,
// skip_empty_lines: true).
func TestParseColumnsMatchesUpstream(t *testing.T) {
	raw, err := os.ReadFile("testdata/csv-parse.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []json.RawMessage
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for i, c := range cases {
		v, _ := dal.DecodeOrderedJSON(c)
		rec := v.(*dal.Record)
		input, _ := rec.Get("input")
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			got, err := ParseColumns(input.(string))
			if wantErr, ok := rec.Get("error"); ok {
				if err == nil || err.Error() != wantErr {
					t.Fatalf("input %q: got error %v, want %q", input, err, wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("input %q: %v", input, err)
			}
			want, _ := rec.Get("records")
			list := make([]any, len(got))
			for i, r := range got {
				list[i] = r
			}
			g, _ := dal.MarshalValue(list)
			w, _ := dal.MarshalValue(want)
			if string(g) != string(w) {
				t.Errorf("input %q:\ngot  %s\nwant %s", input, g, w)
			}
		})
	}
}
