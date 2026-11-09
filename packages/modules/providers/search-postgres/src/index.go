// Package searchpostgres mirrors @medusajs/search-postgres.
package searchpostgres

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/providers/search-postgres/src/migrations"
)

// Module is the module provider definition.
var Module = modulessdk.ModuleExports{
	Migrations: migrations.Migrations,
}
