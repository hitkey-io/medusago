package configexamples

import (
	"os"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
)

// AdminExample sets every admin option.
func AdminExample() config.InputConfig {
	// region: admin
	return config.InputConfig{
		Admin: config.AdminInput{
			Disable:           new(false),
			Path:              new("/app"),
			BackendURL:        new(os.Getenv("MEDUSA_BACKEND_URL")),
			StorefrontURL:     new("https://storefront.example.com"),
			MaxUploadFileSize: new(float64(5 * 1024 * 1024)),
		},
	}
	// endregion: admin
}

// AdminDisabledByEnvironment disables the admin in the worker deployment
// only, based on an environment variable.
func AdminDisabledByEnvironment() config.InputConfig {
	// region: admin-disable
	return config.InputConfig{
		Admin: config.AdminInput{
			Disable: new(os.Getenv("DISABLE_MEDUSA_ADMIN") == "true"),
		},
	}
	// endregion: admin-disable
}
