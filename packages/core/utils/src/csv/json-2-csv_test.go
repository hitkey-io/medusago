package csv

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// testdata/json2csv.json holds json-2-csv 5.6.0 outputs for each input and
// option set.
func TestJson2CsvMatchesUpstream(t *testing.T) {
	raw, err := os.ReadFile("testdata/json2csv.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []json.RawMessage
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for i, c := range cases {
		v, err := dal.DecodeOrderedJSON(c)
		if err != nil {
			t.Fatal(err)
		}
		rec := v.(*dal.Record)
		input, _ := rec.Get("input")
		optsV, _ := rec.Get("options")
		want, _ := rec.Get("output")
		opts := DefaultJson2CsvOptions()
		o := optsV.(*dal.Record)
		for _, k := range o.Keys() {
			val, _ := o.Get(k)
			switch k {
			case "prependHeader":
				opts.PrependHeader = val.(bool)
			case "arrayIndexesAsKeys":
				opts.ArrayIndexesAsKeys = val.(bool)
			case "expandNestedObjects":
				opts.ExpandNestedObjects = val.(bool)
			case "expandArrayObjects":
				opts.ExpandArrayObjects = val.(bool)
			case "preventCsvInjection":
				opts.PreventCsvInjection = val.(bool)
			case "sortHeader":
				opts.SortHeader = val.(bool)
			case "emptyFieldValue":
				s := val.(string)
				opts.EmptyFieldValue = &s
			case "keys":
				for _, key := range val.([]any) {
					opts.Keys = append(opts.Keys, key.(string))
				}
			case "unwindArrays":
			default:
				t.Fatalf("unsupported option %s", k)
			}
		}
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			got := Json2Csv(input.([]any), opts)
			if got != want {
				t.Errorf("options %s\ngot:\n%q\nwant:\n%q", mustJSON(optsV), got, want)
			}
		})
	}
}

func mustJSON(v any) string {
	b, _ := dal.MarshalValue(v)
	return string(b)
}
