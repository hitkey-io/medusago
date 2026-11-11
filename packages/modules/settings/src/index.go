// Package settings mirrors @medusajs/settings.
package settings

import (
	"context"

	coremodulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/settings/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/settings/src/models"
	"github.com/hitkey-io/medusago/packages/modules/settings/src/services"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations:   migrations.Migrations,
	Models:       models.All,
	JoinerConfig: JoinerConfig,
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		return services.NewSettingsModuleService(opts, JoinerConfig)
	},
	// __hooks.onApplicationStart: entity discovery over the joiner configs
	// of the loaded modules (MedusaModule.getAllJoinerConfigs()).
	OnApplicationStart: func(_ context.Context, service any) error {
		if s := services.Of(service); s != nil {
			s.InitializeEntityDiscovery(coremodulessdk.GetAllJoinerConfigs())
		}
		return nil
	},
}
