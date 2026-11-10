// Command medusa-cli mirrors @medusajs/cli: the `medusa` command line tool.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/medusa/src/commands"
	"github.com/hitkey-io/medusago/packages/medusa/src/commands/db"
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: medusa <command> [options]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  start        Start the Medusa server")
	fmt.Fprintln(os.Stderr, "  db:migrate   Run the module migrations")
	fmt.Fprintln(os.Stderr, "  user         Create a user")
}

// loadConfig mirrors the config loader: medusa-config(.json) of the working
// directory passed through defineConfig.
func loadConfig() config.ConfigModule {
	cwd, _ := os.Getwd()
	input, path, err := config.GetConfigFile(cwd, "medusa-config")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load %s: %v\n", path, err)
		os.Exit(1)
	}
	return config.DefineConfig(input)
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

		opts.Config = loadConfig()
		if err := commands.Start(opts); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "db:migrate":
		fs := flag.NewFlagSet("db:migrate", flag.ExitOnError)
		opts := db.MigrateOptions{}
		fs.BoolVar(&opts.SkipLinks, "skip-links", false, "Do not sync links")
		fs.BoolVar(&opts.SkipScripts, "skip-scripts", false, "Do not run migration scripts")
		fs.BoolVar(&opts.ExecuteAllLinks, "execute-all-links", false, "Apply all link changes, including dropping tables of removed links")
		fs.BoolVar(&opts.ExecuteSafeLinks, "execute-safe-links", false, "Apply only safe link changes")
		_ = fs.Parse(os.Args[2:])

		cfg := loadConfig()
		if err := db.Migrate(context.Background(), cfg, opts); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "user":
		fs := flag.NewFlagSet("user", flag.ExitOnError)
		opts := commands.UserOptions{}
		fs.StringVar(&opts.Email, "e", "", "User's email.")
		fs.StringVar(&opts.Email, "email", "", "User's email.")
		fs.StringVar(&opts.Password, "p", "", "User's password.")
		fs.StringVar(&opts.Password, "password", "", "User's password.")
		fs.StringVar(&opts.ID, "i", "", "User's id.")
		fs.StringVar(&opts.ID, "id", "", "User's id.")
		fs.BoolVar(&opts.Invite, "invite", false, "If flag is set, an invitation will be created instead of a new user and the invite token will be returned.")
		_ = fs.Parse(os.Args[2:])

		opts.Config = loadConfig()
		if err := commands.User(opts); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(1)
	}
}
