package services

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Vectors captured from upstream by scripts/extract-provider-vectors.js:
// the provider's send() against a local server recording the SendGrid API
// requests.
type sendgridVectors struct {
	Options map[string]any `json:"options"`
	Cases   []struct {
		Name   string          `json:"name"`
		Input  json.RawMessage `json:"input"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
		Requests []struct {
			Method  string            `json:"method"`
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
			Body    json.RawMessage   `json:"body"`
		} `json:"requests"`
	} `json:"cases"`
}

type recorded struct {
	method, url string
	headers     http.Header
	body        []byte
}

func TestSendMatchesUpstream(t *testing.T) {
	raw, err := os.ReadFile("testdata/upstream-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v sendgridVectors
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	var got []recorded
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = append(got, recorded{r.Method, r.URL.RequestURI(), r.Header.Clone(), body})
		var parsed struct {
			Subject string `json:"subject"`
		}
		_ = json.Unmarshal(body, &parsed)
		switch parsed.Subject {
		case "fail":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"errors":[{"message":"The from address does not match a verified Sender Identity.","field":"from","help":null}]}`))
		case "fail-empty":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(202)
		}
	}))
	defer srv.Close()
	target, _ := url.Parse(srv.URL)

	s := NewSendgridNotificationService(v.Options, nil)
	s.httpClient = &http.Client{Transport: rewrite{target}}
	for _, c := range v.Cases {
		got = nil
		var input *dal.Record
		if string(c.Input) != "null" {
			dv, err := dal.DecodeOrderedJSON(c.Input)
			if err != nil {
				t.Fatal(err)
			}
			input = dv.(*dal.Record)
		}
		res, err := s.Send(input)
		if c.Error != nil {
			var me *common.MedusaError
			if !errors.As(err, &me) || me.Type != c.Error.Type || me.Message != c.Error.Message {
				t.Errorf("%s: error %v, want %s %q", c.Name, err, c.Error.Type, c.Error.Message)
			}
		} else if err != nil || res == nil || len(res.Keys()) != 0 {
			t.Errorf("%s: %v %v", c.Name, res, err)
		}
		if len(got) != len(c.Requests) {
			t.Fatalf("%s: %d requests, want %d", c.Name, len(got), len(c.Requests))
		}
		for i, want := range c.Requests {
			g := got[i]
			if g.method != want.Method || g.url != want.URL {
				t.Errorf("%s: %s %s, want %s %s", c.Name, g.method, g.url, want.Method, want.URL)
			}
			for h, wv := range want.Headers {
				if gv := g.headers.Get(h); gv != wv {
					t.Errorf("%s: header %s = %q, want %q", c.Name, h, gv, wv)
				}
			}
			wb, _ := dal.DecodeOrderedJSON(want.Body)
			wantBody, _ := dal.MarshalValue(wb)
			if string(g.body) != string(wantBody) {
				t.Errorf("%s: body\n got %s\nwant %s", c.Name, g.body, wantBody)
			}
		}
	}
}

type rewrite struct{ target *url.URL }

func (r rewrite) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme, req.URL.Host = r.target.Scheme, r.target.Host
	return http.DefaultTransport.RoundTrip(req)
}
