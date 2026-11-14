// The MedusaGo application of a project: the Medusa CLI running this
// project's configuration and customisations.
package main

// region: main
import (
	"os"

	cli "github.com/hitkey-io/medusago/packages/cli/medusa-cli/src"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.Options{
		Config: medusaConfig,
	}))
}

// endregion: main
