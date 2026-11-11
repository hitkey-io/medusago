// Package loaders mirrors @medusajs/caching/src/loaders.
package loaders

import (
	"fmt"

	"github.com/cespare/xxhash/v2"
)

// Hasher mirrors the hash loader: xxhash-wasm's h64ToString (XXH64, seed
// 0, over the UTF-8 bytes, 16 hex digits).
func Hasher(data string) string {
	return fmt.Sprintf("%016x", xxhash.Sum64String(data))
}
