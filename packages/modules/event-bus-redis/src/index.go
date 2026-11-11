// Package eventbusredis mirrors @medusajs/event-bus-redis.
package eventbusredis

import (
	"context"

	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/event-bus-redis/src/services"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		return services.NewRedisEventBusService(context.Background(), opts.Logger, services.OptionsFromMap(opts.Options))
	},
}
