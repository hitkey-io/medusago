// Package index mirrors @medusajs/index.
package index

import (
	"context"

	coremodulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/index/src/migrations"
	"github.com/hitkey-io/medusago/packages/modules/index/src/models"
	"github.com/hitkey-io/medusago/packages/modules/index/src/services"
	"github.com/hitkey-io/medusago/packages/modules/index/src/utils"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Migrations:   migrations.Migrations,
	Models:       models.All,
	JoinerConfig: JoinerConfig,
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		return services.NewIndexModuleService(opts)
	},
	// `db:migrate` boots the application in server worker mode: the index
	// module's onApplicationStart creates the partitions (no
	// synchronisation).
	Loaders: []modulessdk.Loader{createPartitionsLoader},
}

func createPartitionsLoader(ctx context.Context, opts modulessdk.LoaderOptions) error {
	schema, err := utils.BuildSchemaObjectRepresentation(utils.BaseGraphqlSchema+services.SchemaOption(opts.Options), coremodulessdk.GetAllJoinerConfigs(), opts.Logger)
	if err != nil {
		return err
	}
	return utils.CreatePartitions(ctx, opts.DB, schema, opts.Logger)
}
