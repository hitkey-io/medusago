// Package workflowengineredis mirrors @medusajs/workflow-engine-redis.
package workflowengineredis

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/workflow-engine-redis/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/workflow-engine-redis/src/models"
	"github.com/hitkey-io/medusago/packages/modules/workflow-engine-redis/src/services"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations:   migrations.Migrations,
	Models:       models.All,
	JoinerConfig: JoinerConfig,
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		return services.NewWorkflowsModuleService(opts, JoinerConfig)
	},
}
