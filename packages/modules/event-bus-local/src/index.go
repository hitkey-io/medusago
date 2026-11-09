// Package eventbuslocal mirrors @medusajs/event-bus-local.
package eventbuslocal

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/event-bus-local/src/services"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		return services.NewLocalEventBusService(opts.Logger), nil
	},
}
