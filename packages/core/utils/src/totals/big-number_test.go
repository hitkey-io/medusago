package totals

import (
	"encoding/json"
	"os"
	"testing"
)

// testdata/big-number-vectors.json was produced with the upstream BigNumber:
// [input, raw.value, trimZeros(raw.value), JSON.stringify(bigNumber)].
func TestBigNumberMatchesUpstream(t *testing.T) {
	data, err := os.ReadFile("testdata/big-number-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors [][4]any
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}

	for _, v := range vectors {
		b, err := NewBigNumber(v[0])
		if err != nil {
			t.Errorf("%v: %v", v[0], err)
			continue
		}
		if got := b.Raw().Value; got != v[1] {
			t.Errorf("%#v raw: got %q, want %q", v[0], got, v[1])
		}
		if got := b.TrimmedRaw().Value; got != v[2] {
			t.Errorf("%#v trimmed: got %q, want %q", v[0], got, v[2])
		}
		out, _ := json.Marshal(b)
		if string(out) != v[3] {
			t.Errorf("%#v json: got %s, want %s", v[0], out, v[3])
		}
	}
}

func TestJSNumberToPrecision(t *testing.T) {
	// Values produced by Number(x).toPrecision(20) in Node.js.
	cases := map[float64]string{
		3:     "3.0000000000000000000",
		0.1:   "0.10000000000000000555",
		12.5:  "12.500000000000000000",
		0:     "0.0000000000000000000",
		1e21:  "1.0000000000000000000e+21",
		-2.25: "-2.2500000000000000000",
	}
	for in, want := range cases {
		if got := JSNumberToPrecision(in, 20); got != want {
			t.Errorf("JSNumberToPrecision(%v) = %q, want %q", in, got, want)
		}
	}
}
