package blog

// region: imports
import (
	typesmodulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/www/examples/modules/blog/migrations"
)

// endregion: imports

// Name is the key of the module in the container and in the configuration.
const Name = "blog"

// region: module
// Module is the definition of the blog module.
var Module = utilsmodulessdk.Module(Name, utilsmodulessdk.ModuleDefinition{
	Models:     Models,
	Migrations: migrations.Migrations,
	Loaders:    []typesmodulessdk.Loader{SeedTags},
	Service: func(opts typesmodulessdk.ServiceOptions, base *utilsmodulessdk.MedusaService) (any, error) {
		options, err := ParseOptions(opts.Options)
		if err != nil {
			return nil, err
		}
		return &BlogModuleService{MedusaService: base, options: options}, nil
	},
})

// endregion: module

// region: loaders-on-start
func init() {
	// Run the loaders every time the application starts, not only with
	// `db:migrate`.
	Module.LoadersOnStart = true
}

// endregion: loaders-on-start
