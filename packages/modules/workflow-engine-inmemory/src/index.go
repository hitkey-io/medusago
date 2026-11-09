// Package workflowengineinmemory mirrors @medusajs/workflow-engine-inmemory.
package workflowengineinmemory

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/workflow-engine-inmemory/src/migrations"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations: migrations.Migrations,
}
