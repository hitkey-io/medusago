// Package translation mirrors @medusajs/translation.
package translation

import (
	"context"
	"fmt"

	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/translation/src/loaders"
	"github.com/hitkey-io/medusago/packages/modules/translation/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/translation/src/models"
	"github.com/hitkey-io/medusago/packages/modules/translation/src/services"
)

// TRANSLATION_MODULE mirrors TRANSLATION_MODULE.
const TRANSLATION_MODULE = "translation"

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations:   migrations.Migrations,
	Models:       models.All,
	JoinerConfig: JoinerConfig,
	Loaders:      []modulessdk.Loader{loaders.Defaults},
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		return services.NewTranslationModuleService(opts, JoinerConfig)
	},
	LoadersOnStart: true,
	OnApplicationStart: func(ctx context.Context, service any) error {
		svc, ok := service.(*services.TranslationModuleService)
		if !ok {
			return fmt.Errorf("translation: unexpected service %T", service)
		}
		return svc.OnApplicationStart(ctx)
	},
}
