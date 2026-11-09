// Package modulessdk mirrors the module-related types of
// @medusajs/types/src/modules-sdk.
package modulessdk

import (
	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
	"github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
)

// ModuleExports is what a module or module provider package exports.
type ModuleExports struct {
	// Migrations are the database migrations of the package, in order.
	Migrations []migrations.Migration
	// Models are the data models of the module.
	Models []*dml.Entity
	// JoinerConfig is the joiner configuration of the module service.
	JoinerConfig *joiner.ModuleJoinerConfig
}
