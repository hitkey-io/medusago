// Command medusa-cli mirrors @medusajs/cli: the `medusa` command line tool.
//
// It is the CLI library (packages/cli/medusa-cli/src) without project hooks:
// a project configured with medusa-config.json (or .yaml) only. Go projects call
// cli.Run from their own main package instead.
package main

import (
	"os"

	cli "github.com/hitkey-io/medusago/packages/cli/medusa-cli/src"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.Options{}))
}
