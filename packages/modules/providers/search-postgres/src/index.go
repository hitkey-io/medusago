// Package searchpostgres mirrors @medusajs/search-postgres.
package searchpostgres

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/modules/providers/search-postgres/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/providers/search-postgres/src/services"
)

// Module is the module provider definition. The provider reads and writes
// through the application's database connection (the module's manager
// upstream).
var Module = modulessdk.ModuleExports{
	Migrations:         migrations.Migrations,
	ProviderIdentifier: services.Identifier,
	NewProvider: func(opts modulessdk.ProviderOptions) (any, error) {
		var db dal.Querier
		if opts.Container != nil {
			db, _ = opts.Container.Resolve(common.ContainerRegistrationKeys.PG_CONNECTION).(dal.Querier)
		}
		o := services.Options{}
		o.Language, _ = opts.Options["language"].(string)
		o.Engine, _ = opts.Options["engine"].(string)
		return services.NewPostgresSearchService(db, opts.Logger, o)
	},
}
