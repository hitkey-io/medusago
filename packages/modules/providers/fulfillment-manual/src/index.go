// Package fulfillmentmanual mirrors @medusajs/fulfillment-manual.
package fulfillmentmanual

import modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"

// Module is the module provider definition.
var Module = modulessdk.ModuleExports{
	ProviderIdentifier: "manual",
}
