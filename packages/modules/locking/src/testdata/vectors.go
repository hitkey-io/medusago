// Package testdata loads the lock vectors captured from upstream
// (scripts/extract-lock-vectors.js).
package testdata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// LockKey is one captured key.
type LockKey struct {
	Key                  string `json:"key"`
	AdvisoryLockKey      uint32 `json:"advisory_lock_key"`
	RedisKey             string `json:"redis_key"`
	RedisKeyCustomPrefix string `json:"redis_key_custom_prefix"`
}

// Script is a Lua command defined by locking-redis.
type Script struct {
	NumberOfKeys int    `json:"numberOfKeys"`
	Lua          string `json:"lua"`
}

// Vectors are the captured vectors.
type Vectors struct {
	Keys    []LockKey         `json:"keys"`
	Scripts map[string]Script `json:"scripts"`
}

// Load reads upstream-lock-keys.json.
func Load() (*Vectors, error) {
	_, file, _, _ := runtime.Caller(0)
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(file), "upstream-lock-keys.json"))
	if err != nil {
		return nil, err
	}
	var v Vectors
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
