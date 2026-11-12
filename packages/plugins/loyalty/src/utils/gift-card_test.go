package utils

import (
	"strings"
	"testing"
	"time"
)

// Mirrors utils/__tests__/gift-card.spec.ts.
func TestIsGiftCardExpired(t *testing.T) {
	iso := func(d time.Time) string { return d.UTC().Format("2006-01-02T15:04:05.000Z") }

	t.Run("returns false when the gift card has no expiration date", func(t *testing.T) {
		if IsGiftCardExpired(nil) {
			t.Fatal("expected false")
		}
	})
	t.Run("returns false when the expiration date is in the future", func(t *testing.T) {
		if IsGiftCardExpired(iso(time.Now().Add(time.Hour))) {
			t.Fatal("expected false")
		}
	})
	t.Run("returns true when the expiration date is in the past", func(t *testing.T) {
		if !IsGiftCardExpired(iso(time.Now().Add(-time.Hour))) {
			t.Fatal("expected true")
		}
	})
	t.Run("compares expiration dates in UTC regardless of the input's offset", func(t *testing.T) {
		expiresAt := strings.Replace(iso(time.Now().Add(-time.Hour)), "Z", "+00:00", 1)
		if !IsGiftCardExpired(expiresAt) {
			t.Fatal("expected true")
		}
	})
}
