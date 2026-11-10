// Command create-medusago-app mirrors create-medusa-app: it creates a new
// MedusaGo project (backend, admin and optional Next.js storefront), its
// database and an admin invite, then starts the application.
//
//	go run github.com/hitkey-io/medusago/packages/cli/create-medusago-app@latest my-medusa-store
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	fs := flag.NewFlagSet("create-medusago-app", flag.ExitOnError)
	opts := ProjectOptions{Browser: true, Migrations: true}
	fs.StringVar(&opts.RepoURL, "repo-url", "", "URL of repository to use to setup project.")
	fs.StringVar(&opts.Version, "version", "", "The version of MedusaGo to install.")
	fs.BoolVar(&opts.Seed, "seed", false, "Seed the created database with demo data.")
	fs.BoolVar(&opts.SkipDB, "skip-db", false, "Skips creating the database, running migrations, and seeding, and subsequently skips opening the browser.")
	fs.StringVar(&opts.DBURL, "db-url", "", "Skips database creation and sets the database URL to the provided URL. Throws an error if can't connect to the database. Will still run migrations and open the admin after project creation.")
	noMigrations := fs.Bool("no-migrations", false, "Skips running migrations, creating admin user, and seeding. If used, it's expected that you pass the --db-url option with a url of a database that has all necessary migrations. Otherwise, unexpected errors will occur.")
	noBrowser := fs.Bool("no-browser", false, "Disables opening the browser at the end of the project creation and only shows success message.")
	fs.StringVar(&opts.DirectoryPath, "directory-path", "", "Specify the directory path to install the project in.")
	fs.BoolVar(&opts.WithNextjsStarter, "with-nextjs-starter", false, "Install the storefront along with the Medusa backend")
	fs.BoolVar(&opts.Verbose, "verbose", false, "Show all logs of underlying commands. Useful for debugging.")
	useNpm := fs.Bool("use-npm", false, "Use npm as the package manager of the storefront")
	useYarn := fs.Bool("use-yarn", false, "Use yarn as the package manager of the storefront")
	usePnpm := fs.Bool("use-pnpm", false, "Use pnpm as the package manager of the storefront")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: create-medusago-app [options] [project-name]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Create a new MedusaGo project")
		fmt.Fprintln(os.Stderr)
		fs.PrintDefaults()
	}
	// flags may follow the project name, like commander
	var args []string
	rest := os.Args[1:]
	for len(rest) > 0 {
		if err := fs.Parse(rest); err != nil {
			os.Exit(1)
		}
		rest = fs.Args()
		if len(rest) > 0 {
			args = append(args, rest[0])
			rest = rest[1:]
		}
	}
	opts.Migrations = !*noMigrations
	opts.Browser = !*noBrowser
	set := 0
	for _, b := range []bool{*useNpm, *useYarn, *usePnpm} {
		if b {
			set++
		}
	}
	if set > 1 {
		fmt.Fprintln(os.Stderr, "error: options --use-npm, --use-yarn and --use-pnpm cannot be used together")
		os.Exit(1)
	}
	switch {
	case *useYarn:
		opts.PackageManager = "yarn"
	case *usePnpm:
		opts.PackageManager = "pnpm"
	case *useNpm:
		opts.PackageManager = "npm"
	}

	creator, err := newProjectCreator(args, opts)
	if err != nil {
		logError(err.Error())
	}
	creator.create()
}
