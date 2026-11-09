// Package filelocal mirrors @medusajs/file-local.
package filelocal

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/providers/file-local/src/services"
)

// Module is the module provider definition.
var Module = modulessdk.ModuleExports{
	ProviderIdentifier: "localfs",
	NewProvider: func(opts modulessdk.ProviderOptions) (any, error) {
		return services.NewLocalFileService(opts.Options), nil
	},
}
