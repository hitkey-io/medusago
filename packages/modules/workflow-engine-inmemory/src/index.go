// Package workflowengineinmemory mirrors @medusajs/workflow-engine-inmemory.
package workflowengineinmemory

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/workflow-engine-inmemory/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/workflow-engine-inmemory/src/models"
	"github.com/hitkey-io/medusago/packages/modules/workflow-engine-inmemory/src/services"
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
