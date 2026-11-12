// Package utils mirrors src/utils of @medusajs/loyalty-plugin.
package utils

import (
	"fmt"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// IsGiftCardExpired mirrors isGiftCardExpired({ expires_at }): a gift card
// without an expiration date never expires; otherwise the expiration date
// is compared with the current time (both as UTC timestamps).
func IsGiftCardExpired(expiresAt any) bool {
	return IsGiftCardExpiredAt(expiresAt, time.Now())
}

// IsGiftCardExpiredAt is IsGiftCardExpired with the current time given.
func IsGiftCardExpiredAt(expiresAt any, now time.Time) bool {
	var t time.Time
	switch v := expiresAt.(type) {
	case nil:
		return false
	case string:
		if v == "" {
			return false
		}
		parsed, ok := common.ParseJSDate(v)
		if !ok {
			// new Date(invalid).getTime() is NaN: NaN <= now is false
			return false
		}
		t = parsed
	case time.Time:
		t = v
	case *time.Time:
		if v == nil {
			return false
		}
		t = *v
	case float64:
		if v == 0 {
			return false
		}
		t = time.UnixMilli(int64(v))
	default:
		parsed, ok := common.ParseJSDate(fmt.Sprint(v))
		if !ok {
			return false
		}
		t = parsed
	}
	return t.UnixMilli() <= now.UnixMilli()
}
