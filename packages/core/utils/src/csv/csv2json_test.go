package csv

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func undefinedAsMarker(v any) any {
	switch t := v.(type) {
	case undefinedValue:
		return "__undefined__"
	case *dal.Record:
		out := dal.NewRecord()
		for _, k := range t.Keys() {
			x, _ := t.Get(k)
			out.Set(k, undefinedAsMarker(x))
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = undefinedAsMarker(x)
		}
		return out
	}
	return v
}

// testdata/csv2json.json holds json-2-csv 5.6.0 csv2json outputs.
func TestCsv2JsonMatchesUpstream(t *testing.T) {
	raw, err := os.ReadFile("testdata/csv2json.json")
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
		field, _ := rec.Get("field")
		want, _ := rec.Get("output")
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			got := Csv2Json(input.(string), field.(string))
			list := make([]any, len(got))
			for i, r := range got {
				list[i] = undefinedAsMarker(r)
			}
			g, _ := dal.MarshalValue(list)
			w, _ := dal.MarshalValue(want)
			if string(g) != string(w) {
				t.Errorf("input %q:\ngot  %s\nwant %s", input, g, w)
			}
		})
	}
}
