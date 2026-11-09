package commands

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	srv := httptest.NewServer(NewApp())
	defer srv.Close()

	res, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || string(body) != "OK" {
		t.Fatalf("got %d %q, want 200 \"OK\"", res.StatusCode, body)
	}
}
