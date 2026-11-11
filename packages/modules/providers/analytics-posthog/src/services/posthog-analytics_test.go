package services

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Vectors captured from upstream by scripts/extract-provider-vectors.js:
// track/identify calls, then shutdown, against a local server recording
// the PostHog batch requests.
type posthogVectors struct {
	Calls []struct {
		Method string          `json:"method"`
		Data   json.RawMessage `json:"data"`
		Result string          `json:"result"`
		Error  string          `json:"error"`
	} `json:"calls"`
	Requests []struct {
		Method  string            `json:"method"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
		Body    json.RawMessage   `json:"body"`
	} `json:"requests"`
	ConstructorError string `json:"constructor_error"`
}

var (
	isoRe  = regexp.MustCompile(`"\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\.\d{3}Z"`)
	uuidRe = regexp.MustCompile(`"[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}"`)
)

func mask(b []byte) string {
	return uuidRe.ReplaceAllString(isoRe.ReplaceAllString(string(b), `"<date>"`), `"<uuid>"`)
}

func TestPosthogMatchesUpstream(t *testing.T) {
	raw, err := os.ReadFile("testdata/upstream-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v posthogVectors
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	type req struct {
		method, url string
		header      http.Header
		body        []byte
	}
	var got []req
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body io.Reader = r.Body
		if r.Header.Get("Content-Encoding") == "gzip" {
			zr, err := gzip.NewReader(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			body = zr
		}
		b, _ := io.ReadAll(body)
		got = append(got, req{r.Method, r.URL.RequestURI(), r.Header.Clone(), b})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	if _, err := NewPosthogAnalyticsService(map[string]any{}, nil); err == nil || err.Error() != v.ConstructorError {
		t.Errorf("constructor error %v, want %q", err, v.ConstructorError)
	}
	s, err := NewPosthogAnalyticsService(map[string]any{"posthogEventsKey": "phc_test", "posthogHost": srv.URL + "/"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, c := range v.Calls {
		dv, _ := dal.DecodeOrderedJSON(c.Data)
		data := dv.(*dal.Record)
		var err error
		if c.Method == "track" {
			err = s.Track(ctx, data)
		} else {
			err = s.Identify(ctx, data)
		}
		gotErr := ""
		if err != nil {
			gotErr = err.Error()
		}
		if gotErr != c.Error {
			t.Errorf("%s %s: error %q, want %q", c.Method, c.Data, gotErr, c.Error)
		}
	}
	if err := s.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(v.Requests) {
		t.Fatalf("%d requests, want %d", len(got), len(v.Requests))
	}
	for i, want := range v.Requests {
		g := got[i]
		if g.method != want.Method || g.url != want.URL {
			t.Errorf("request %s %s, want %s %s", g.method, g.url, want.Method, want.URL)
		}
		for h, wv := range want.Headers {
			if gv := g.header.Get(h); gv != wv {
				t.Errorf("header %s = %q, want %q", h, gv, wv)
			}
		}
		wb, _ := dal.DecodeOrderedJSON(want.Body)
		wantBody, _ := dal.MarshalValue(wb)
		if mask(g.body) != mask(wantBody) {
			t.Errorf("body\n got %s\nwant %s", mask(g.body), mask(wantBody))
		}
	}
}
