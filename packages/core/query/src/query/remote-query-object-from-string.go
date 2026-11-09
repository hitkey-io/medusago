package query

import (
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// RemoteQueryObjectConfig mirrors RemoteQueryObjectConfig.
type RemoteQueryObjectConfig struct {
	EntryPoint string
	// Variables become the root args, except "<relation path>" keys which
	// become the args of that relation.
	Variables *dal.Record
	Fields    []string
}

// RemoteQueryObjectFromString mirrors remoteQueryObjectFromString (utils/common
// upstream; it lives here because dal imports common). The
// result uses the "__fields" / "__args" keys of the remote query object.
func RemoteQueryObjectFromString(c RemoteQueryObjectConfig) *dal.Record {
	entry := dal.NewRecord()
	entry.Set("__fields", []string{})
	used := map[string]bool{}
	child := func(r *dal.Record, key string) *dal.Record {
		if v, ok := r.Get(key); ok {
			if x, ok := v.(*dal.Record); ok {
				return x
			}
		}
		x := dal.NewRecord()
		r.Set(key, x)
		return x
	}
	addField := func(r *dal.Record, f string) {
		v, _ := r.Get("__fields")
		list, _ := v.([]string)
		r.Set("__fields", append(list, f))
	}
	for _, field := range c.Fields {
		if !strings.Contains(field, ".") {
			addField(entry, field)
			continue
		}
		segs := strings.Split(field, ".")
		prop := segs[len(segs)-1]
		ref := entry
		combined := ""
		for _, seg := range segs[:len(segs)-1] {
			if combined == "" {
				combined = seg
			} else {
				combined += "." + seg
			}
			ref = child(ref, seg)
			if c.Variables != nil {
				if v, ok := c.Variables.Get(combined); ok {
					ref.Set("__args", v)
					used[combined] = true
				}
			}
		}
		addField(ref, prop)
	}
	args := dal.NewRecord()
	if c.Variables != nil {
		for _, k := range c.Variables.Keys() {
			if !used[k] {
				v, _ := c.Variables.Get(k)
				args.Set(k, v)
			}
		}
	}
	entry.Set("__args", args)
	out := dal.NewRecord()
	out.Set(c.EntryPoint, entry)
	return out
}
