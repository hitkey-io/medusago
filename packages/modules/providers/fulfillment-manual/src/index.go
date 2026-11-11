// Package fulfillmentmanual mirrors @medusajs/fulfillment-manual.
package fulfillmentmanual

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/providers/fulfillment-manual/src/services"
)

// Module is the module provider definition.
var Module = modulessdk.ModuleExports{
	ProviderIdentifier: services.Identifier,
	NewProvider: func(modulessdk.ProviderOptions) (any, error) {
		return services.ManualFulfillmentService{}, nil
	},
}
