// Package modulessdk mirrors the module-related types of
// @medusajs/types/src/modules-sdk.
package modulessdk

import "github.com/hitkey-io/medusago/packages/core/utils/src/migrations"

// ModuleExports is what a module or module provider package exports.
type ModuleExports struct {
	// Migrations are the database migrations of the package, in order.
	Migrations []migrations.Migration
}
