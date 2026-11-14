// Command tutorial is the entry point of the MedusaGo application of the
// customisation tutorial: the Brand Module, linked to products.
package main

import (
	"log"
	"os"

	cli "github.com/hitkey-io/medusago/packages/cli/medusa-cli/src"
	"github.com/hitkey-io/medusago/www/examples/tutorial/src/api"
	"github.com/hitkey-io/medusago/www/examples/tutorial/src/links"
	"github.com/hitkey-io/medusago/www/examples/tutorial/src/workflows/hooks"
)

// region: main
func main() {
	// define the module links and consume the workflow hooks before the
	// application starts
	if err := links.Define(); err != nil {
		log.Fatal(err)
	}
	hooks.Register()

	os.Exit(cli.Run(os.Args[1:], cli.Options{
		Config:      medusaConfig,
		Routes:      api.Routes,
		Middlewares: api.Middlewares,
	}))
}

// endregion: main
