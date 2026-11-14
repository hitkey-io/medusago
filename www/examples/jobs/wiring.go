package jobs

import (
	"context"

	cli "github.com/hitkey-io/medusago/packages/cli/medusa-cli/src"
	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
)

// medusaConfig stands for the configuration of your project.
func medusaConfig() config.InputConfig { return config.InputConfig{} }

// region: options
// Options are the options main.go passes to the CLI.
func Options() cli.Options {
	return cli.Options{
		Config: medusaConfig,
		OnLoad: []func(ctx context.Context, app *loaders.MedusaApp) error{
			RegisterJobs,
		},
	}
}

// endregion: options
