package common

import (
	"regexp"
	"testing"
)

func TestGenerateEntityID(t *testing.T) {
	if got := GenerateEntityID("prod_existing", "prod"); got != "prod_existing" {
		t.Fatalf("expected existing id to be returned, got %q", got)
	}

	re := regexp.MustCompile(`^prod_[0-9A-HJKMNP-TV-Z]{26}$`)
	if got := GenerateEntityID("", "prod"); !re.MatchString(got) {
		t.Fatalf("unexpected prefixed id %q", got)
	}

	bare := regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)
	if got := GenerateEntityID("", ""); !bare.MatchString(got) {
		t.Fatalf("unexpected bare id %q", got)
	}
}
