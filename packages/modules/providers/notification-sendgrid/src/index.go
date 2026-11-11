// Package notificationsendgrid mirrors @medusajs/notification-sendgrid.
package notificationsendgrid

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/providers/notification-sendgrid/src/services"
)

// Module is the module provider definition.
var Module = modulessdk.ModuleExports{
	ProviderIdentifier: services.Identifier,
	NewProvider: func(opts modulessdk.ProviderOptions) (any, error) {
		return services.NewSendgridNotificationService(opts.Options, opts.Logger), nil
	},
}
