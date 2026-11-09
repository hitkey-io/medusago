package common

import (
	"strconv"
	"unicode/utf16"
)

// SimpleHash mirrors simpleHash from simple-hash.ts, reproducing the
// JavaScript number semantics: `hash << 5` is a 32-bit signed shift while the
// accumulated hash itself is a float64 that is not truncated.
func SimpleHash(text string) string {
	hash := int64(5381)
	for _, c := range utf16.Encode([]rune(text)) {
		shifted := int64(int32(uint32(int32(uint32(hash))) << 5))
		hash = shifted + hash + int64(c)
	}
	return strconv.FormatInt(hash, 16)
}
