// Package notificationlocal mirrors @medusajs/notification-local.
package notificationlocal

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/providers/notification-local/src/services"
)

// Module is the module provider definition.
var Module = modulessdk.ModuleExports{
	ProviderIdentifier: "notification-local",
	NewProvider: func(opts modulessdk.ProviderOptions) (any, error) {
		return services.NewLocalNotificationService(opts.Logger), nil
	},
}
