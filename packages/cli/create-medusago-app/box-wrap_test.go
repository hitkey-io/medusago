package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestBoxWrapsToTerminalWidth(t *testing.T) {
	t.Setenv("COLUMNS", "60")
	msg := "After you start the Medusa application, you can create an admin user with the URL http://localhost:9000/app/invite?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6Imludml0ZV8wMUoiLCJlbWFpbCI6ImFkbWluQGV4YW1wbGUudGVzdCJ9.c2lnbmF0dXJlc2lnbmF0dXJl&first_run=true\n\nshort"
	out := box(msg, "")
	for _, l := range strings.Split(out, "\n") {
		if n := utf8.RuneCountInString(l); n > 60 {
			t.Fatalf("line wider than the terminal (%d): %q", n, l)
		}
	}
	joined := strings.NewReplacer("│", "", " ", "", "\n", "").Replace(out)
	if !strings.Contains(joined, "first_run=true") || !strings.Contains(joined, "short") {
		t.Fatalf("text lost:\n%s", out)
	}
}
