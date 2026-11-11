// Package file mirrors @medusajs/file.
package file

import (
	"errors"

	filetypes "github.com/hitkey-io/medusago/packages/core/types/src/file"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/file/src/services"
	localfile "github.com/hitkey-io/medusago/packages/modules/providers/file-local/src/services"
)

// Module is the module definition: the service uses the configured file
// provider (exactly one, like loadProviders).
var Module = modulessdk.ModuleExports{
	JoinerConfig: JoinerConfig,
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		if len(opts.Providers) > 1 {
			return nil, errors.New("File module: You can only configure one file provider")
		}
		var provider filetypes.IFileProvider
		if len(opts.Providers) == 1 {
			provider, _ = opts.Providers[0].Instance.(filetypes.IFileProvider)
		}
		if provider == nil {
			provider = localfile.NewLocalFileService(nil)
		}
		return services.NewFileModuleService(provider), nil
	},
}
