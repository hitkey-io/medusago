// Package cacheinmemory mirrors @medusajs/cache-inmemory: the default cache
// module (Modules.CACHE).
package cacheinmemory

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/cache-inmemory/src/services"
)

// numberOption reads a numeric module option (nil when absent).
func numberOption(opts map[string]any, key string) *float64 {
	switch v := opts[key].(type) {
	case float64:
		return &v
	case int:
		f := float64(v)
		return &f
	case int64:
		f := float64(v)
		return &f
	}
	return nil
}

// Module is the module definition (options: ttl in seconds, default 30).
var Module = modulessdk.ModuleExports{
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		return services.NewInMemoryCacheService(numberOption(opts.Options, "ttl")), nil
	},
}
