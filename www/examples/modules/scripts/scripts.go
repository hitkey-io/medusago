package scripts

// region: imports
import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	typescommon "github.com/hitkey-io/medusago/packages/core/types/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// endregion: imports

// region: my-script
// MyScript logs the number of products. It is registered with the name
// "my-script".
func MyScript(ctx context.Context, args typescommon.ExecArgs) error {
	logger := args.Container.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)
	q := args.Container.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)

	res, err := q.Graph(ctx, query.GraphInput{
		Entity: "product",
		Fields: []string{"id"},
	}, joiner.Options{})
	if err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("Found %d products", len(res.Data)))
	return nil
}

// endregion: my-script

// region: script-args
// Greet logs the arguments that follow the script name:
// `go run . exec greet Ada Grace`.
func Greet(ctx context.Context, args typescommon.ExecArgs) error {
	logger := args.Container.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)
	for _, name := range args.Args {
		logger.Info("Hello, " + name)
	}
	return nil
}

// endregion: script-args
