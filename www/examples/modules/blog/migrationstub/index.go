// Package migrations lists the migrations of the module. Create it with an
// empty list before the first db:generate: the command rewrites this file.
package migrations

import (
	m "github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
)

var Migrations = []m.Migration{}
