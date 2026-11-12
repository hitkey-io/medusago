// Package steps mirrors src/workflows/gift-cards/steps of
// @medusajs/loyalty-plugin.
package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/loyalty"
)

// loyaltyModule resolves the loyalty module service (PluginModule.LOYALTY).
func loyaltyModule(c composer.Container) *loyalty.LoyaltyModuleService {
	s, _ := c.Resolve(loyalty.ServiceName).(*loyalty.LoyaltyModuleService)
	return s
}

// get returns r[k] (nil when absent).
func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

// str returns r[k] as a string.
func str(r *dal.Record, k string) string {
	s, _ := get(r, k).(string)
	return s
}

// asRecords converts a list of records.
func asRecords(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		out := make([]*dal.Record, 0, len(t))
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	case *dal.Record:
		return []*dal.Record{t}
	}
	return nil
}

func toAny(rs []*dal.Record) []any {
	out := make([]any, len(rs))
	for i, r := range rs {
		out[i] = r
	}
	return out
}

func ids(rs []*dal.Record) []any {
	out := make([]any, len(rs))
	for i, r := range rs {
		out[i] = get(r, "id")
	}
	return out
}
