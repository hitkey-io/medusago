// Command medusa-cli mirrors @medusajs/cli: the `medusa` command line tool.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hitkey-io/medusago/packages/medusa/src/commands"
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: medusa <command> [options]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  start    Start the Medusa server")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "start":
		fs := flag.NewFlagSet("start", flag.ExitOnError)
		opts := commands.StartOptions{}
		fs.StringVar(&opts.Host, "H", "", "Set host")
		fs.StringVar(&opts.Host, "host", "", "Set host")
		fs.IntVar(&opts.Port, "p", 9000, "Set port")
		fs.IntVar(&opts.Port, "port", 9000, "Set port")
		_ = fs.Parse(os.Args[2:])

		if err := commands.Start(opts); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(1)
	}
}
