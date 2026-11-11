// Package lockingpostgres mirrors @medusajs/locking-postgres.
package lockingpostgres

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/modules/providers/locking-postgres/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/providers/locking-postgres/src/services"
)

// Module is the module provider definition.
var Module = modulessdk.ModuleExports{
	Migrations:         migrations.Migrations,
	ProviderIdentifier: services.Identifier,
	NewProvider: func(opts modulessdk.ProviderOptions) (any, error) {
		var pool *pgxpool.Pool
		if opts.Container != nil {
			pool, _ = opts.Container.Resolve(common.ContainerRegistrationKeys.PG_CONNECTION).(*pgxpool.Pool)
		}
		if pool == nil {
			return nil, fmt.Errorf("locking-postgres: no database connection")
		}
		return services.NewPostgresAdvisoryLockProvider(pool), nil
	},
}
