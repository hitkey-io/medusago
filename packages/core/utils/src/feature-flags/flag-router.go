// Package featureflags mirrors @medusajs/utils/src/feature-flags.
package featureflags

import (
	"os"
	"strings"
	"sync"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// FlagSettings mirrors FlagSettings.
type FlagSettings struct {
	Key         string
	DefaultVal  bool
	EnvKey      string
	Description string
}

// FlagRouter mirrors FlagRouter. Values are bool or map[string]bool
// (nested flags), kept in registration order.
type FlagRouter struct {
	mu    sync.RWMutex
	flags *dal.Record
}

// NewFlagRouter creates an empty router.
func NewFlagRouter() *FlagRouter { return &FlagRouter{flags: dal.NewRecord()} }

// IsFeatureEnabled mirrors isFeatureEnabled for a single key.
func (r *FlagRouter) IsFeatureEnabled(key string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, _ := r.flags.Get(key)
	switch t := v.(type) {
	case bool:
		return t
	case nil:
		return false
	}
	return true
}

// IsNestedFeatureEnabled mirrors isFeatureEnabled({ key: nested }).
func (r *FlagRouter) IsNestedFeatureEnabled(key, nested string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, _ := r.flags.Get(key)
	switch t := v.(type) {
	case bool:
		return t
	case *dal.Record:
		x, _ := t.Get(nested)
		b, _ := x.(bool)
		return b
	}
	return false
}

// SetFlag mirrors setFlag.
func (r *FlagRouter) SetFlag(key string, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec, ok := value.(*dal.Record); ok {
		existing, _ := r.flags.Get(key)
		if ex, ok := existing.(*dal.Record); ok {
			for _, k := range rec.Keys() {
				v, _ := rec.Get(k)
				ex.Set(k, v)
			}
			return
		}
		r.flags.Set(key, rec)
		return
	}
	r.flags.Set(key, value)
}

// ListFlags mirrors listFlags: { key: value } in registration order.
func (r *FlagRouter) ListFlags() *dal.Record {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := dal.NewRecord()
	for _, k := range r.flags.Keys() {
		v, _ := r.flags.Get(k)
		out.Set(k, v)
	}
	return out
}

func isTruthy(s string) bool { return strings.ToLower(s) == "true" }

// objectFromStringPath mirrors objectFromStringPath for "a.b" entries.
func objectFromStringPath(paths []string) *dal.Record {
	out := dal.NewRecord()
	for _, p := range paths {
		parts := strings.Split(p, ".")
		cur := out
		for i, part := range parts {
			if i == len(parts)-1 {
				cur.Set(part, true)
				break
			}
			next, _ := cur.Get(part)
			r, ok := next.(*dal.Record)
			if !ok {
				r = dal.NewRecord()
				cur.Set(part, r)
			}
			cur = r
		}
	}
	return out
}

// RegisterFeatureFlag mirrors registerFeatureFlag (environment first, then
// projectConfig.featureFlags).
func RegisterFeatureFlag(flag FlagSettings, projectConfigFlags map[string]any, router *FlagRouter) {
	var value any = flag.DefaultVal
	if env, ok := os.LookupEnv(flag.EnvKey); ok {
		value = isTruthy(env)
		if parts := strings.Split(env, ","); len(parts) > 1 {
			value = objectFromStringPath(parts)
		}
	} else if pc, ok := projectConfigFlags[flag.Key]; ok && pc != nil {
		switch t := pc.(type) {
		case bool:
			value = t
		case string:
			value = isTruthy(t)
		case map[string]any:
			r := dal.NewRecord()
			for k, v := range t {
				b, _ := v.(bool)
				r.Set(k, b)
			}
			value = r
		default:
			value = true
		}
	}
	router.SetFlag(flag.Key, value)
}
