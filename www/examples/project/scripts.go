package main

import (
	"context"
	"log/slog"
	"os"
	"strings"

	cli "github.com/hitkey-io/medusago/packages/cli/medusa-cli/src"
	typescommon "github.com/hitkey-io/medusago/packages/core/types/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// region: hello
// hello is run with `go run . exec hello Medusa`.
func hello(ctx context.Context, args typescommon.ExecArgs) error {
	logger, _ := args.Container.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)
	logger.Info("Hello " + strings.Join(args.Args, " "))
	return nil
}

// endregion: hello

// mainWithScripts is what main looks like when the project has scripts.
func mainWithScripts() {
	// region: register
	os.Exit(cli.Run(os.Args[1:], cli.Options{
		Config: medusaConfig,
		Scripts: map[string]typescommon.ExecScript{
			"hello": hello,
		},
	}))
	// endregion: register
}
