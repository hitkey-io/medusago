// Package cli mirrors @medusajs/cli (create-cli.ts): the `medusa` command
// line interface, as a library.
//
// A Medusa project written in Go is a Go module whose main package calls Run:
//
//	package main
//
//	import (
//		"os"
//
//		cli "github.com/hitkey-io/medusago/packages/cli/medusa-cli/src"
//	)
//
//	func main() {
//		os.Exit(cli.Run(os.Args[1:], cli.Options{}))
//	}
//
// and is used like a Medusa project: `go run . db:setup`, `go run . develop`,
// `go run . build`, `.medusa/server/medusa start`. The stock binary
// (packages/cli/medusa-cli) is the same CLI without project options.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hitkey-io/medusago/packages/cli/medusa-cli/src/reporter"
	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	typescommon "github.com/hitkey-io/medusago/packages/core/types/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/medusa/src/commands"
	"github.com/hitkey-io/medusago/packages/medusa/src/commands/db"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
)

// Version is the Medusa version this CLI mirrors (UPSTREAM_VERSION).
const Version = "2.22.4"

// defaultPort mirrors defaultPort of create-cli.ts.
const defaultPort = "9000"

// Options are the project hooks of the CLI.
type Options struct {
	// Config returns the project configuration (the object passed to
	// defineConfig in medusa-config.ts). It is called after the .env files
	// are loaded, so it may read environment variables. When nil, the
	// medusa-config.json (or .yaml/.yml) file of the project directory is used.
	Config func() config.InputConfig
	// Scripts are the scripts `medusa exec <name>` can run. A script given
	// by file path (./src/scripts/seed.ts) resolves to the name "seed".
	Scripts map[string]typescommon.ExecScript
	// OnLoad hooks run after the application is loaded by start, develop,
	// exec and user (for example to register routes on app.App or
	// subscribers on the event bus).
	OnLoad []func(ctx context.Context, app *loaders.MedusaApp) error
	// Stdout and Stderr receive the CLI help and errors (default os.Stdout
	// and os.Stderr). Logs go through slog.
	Stdout, Stderr io.Writer
}

// command is a CLI command definition.
type command struct {
	name     string
	usage    string
	desc     string
	options  []option
	nodeEnv  string // NODE_ENV default set by the handler ("" leaves it)
	rest     int    // see parseArgs
	required int    // number of required positionals
	run      func(r *runner, a parsedArgs) int
}

type runner struct {
	opts      Options
	directory string
	logger    *slog.Logger
}

// loadConfig mirrors the config loader: medusa-config(.json) passed through
// defineConfig, after the .env files are loaded (loadEnv in medusa-config).
func (r *runner) loadConfig() (config.ConfigModule, error) {
	nodeEnv := os.Getenv("NODE_ENV")
	if nodeEnv == "" {
		nodeEnv = "development"
	}
	common.LoadEnv(nodeEnv, r.directory)
	if r.opts.Config != nil {
		return config.DefineConfig(r.opts.Config()), nil
	}
	input, path, err := config.GetConfigFile(r.directory, "medusa-config")
	if err != nil {
		return config.ConfigModule{}, fmt.Errorf("Failed to load %s: %w", path, err)
	}
	return config.DefineConfig(input), nil
}

func (r *runner) mustConfig() (config.ConfigModule, bool) {
	cfg, err := r.loadConfig()
	if err != nil {
		r.logger.Error(err.Error())
		return cfg, false
	}
	return cfg, true
}

func exitCode(ok bool) int {
	if ok {
		return 0
	}
	return 1
}

// envDefault mirrors `default: process.env.X`.
func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var migrateLinkSearchOptions = []option{
	{name: "skip-links", kind: boolOption, describe: "Do not sync links"},
	{name: "skip-search", kind: boolOption, describe: "Do not create or alter search indexes"},
	{name: "execute-all-links", kind: boolOption, describe: "Skip prompts and execute all (including unsafe) actions from sync links"},
	{name: "execute-safe-links", kind: boolOption, describe: "Skip prompts and execute only safe actions from sync links"},
	{name: "execute-all-search", kind: boolOption, describe: "Skip prompts and execute all (including unsafe) actions on search indexes"},
	{name: "execute-safe-search", kind: boolOption, describe: "Skip prompts and execute only safe actions on search indexes"},
}

var dbCreateOptions = []option{
	{name: "db", kind: stringOption, describe: "Specify the name of the database you want to create"},
	{name: "interactive", kind: boolOption, def: true, describe: "Display prompts. Use --no-interactive flag to run the command without prompts"},
}

var lintOptions = []option{
	{name: "lint", kind: boolOption, def: false, describe: "Run the Medusa linter (not supported by the Go CLI; ignored)."},
	{name: "fix", kind: boolOption, def: false, describe: "Auto-fix lint issues where possible (ignored)."},
}

func migrateOptions(a parsedArgs) db.MigrateOptions {
	concurrency, _ := a.number("concurrency")
	return db.MigrateOptions{
		SkipLinks:         a.bool("skip-links"),
		SkipScripts:       a.bool("skip-scripts"),
		SkipSearch:        a.bool("skip-search"),
		ExecuteAllLinks:   a.bool("execute-all-links"),
		ExecuteSafeLinks:  a.bool("execute-safe-links"),
		ExecuteAllSearch:  a.bool("execute-all-search"),
		ExecuteSafeSearch: a.bool("execute-safe-search"),
		Concurrency:       int(concurrency),
		AllOrNothing:      a.bool("all-or-nothing"),
	}
}

// notSupported reports a command of the Node CLI the Go CLI does not provide.
func notSupported(reason string) func(r *runner, a parsedArgs) int {
	return func(r *runner, _ parsedArgs) int {
		r.logger.Error(reason)
		return 1
	}
}

func (r *runner) startOptions(a parsedArgs, cfg config.ConfigModule) (commands.StartOptions, bool) {
	port, err := strconv.Atoi(a.string("port"))
	if err != nil {
		r.logger.Error(fmt.Sprintf("Invalid port: %s", a.string("port")))
		return commands.StartOptions{}, false
	}
	return commands.StartOptions{
		Host:      a.string("host"),
		Port:      port,
		Config:    cfg,
		Directory: r.directory,
		OnLoad:    r.opts.OnLoad,
	}, true
}

func commandsList() []command {
	hostOption := option{name: "H", aliases: []string{"host"}, kind: stringOption, def: os.Getenv("HOST"), describe: "Set host"}
	portOption := option{name: "p", aliases: []string{"port"}, kind: stringOption, def: envDefault("PORT", defaultPort), describe: "Set port. Defaults to " + defaultPort}
	return []command{
		{
			name: "new", usage: "new [root] [starter]", desc: "Create a new Medusa project.",
			run: notSupported("The new command is not available in the Go CLI. Use the Go port of create-medusa-app to create a project."),
		},
		{
			name: "db:setup", desc: "Create the database, run migrations and sync links",
			options: append(append([]option{}, dbCreateOptions...), migrateLinkSearchOptions...),
			nodeEnv: "development",
			run: func(r *runner, a parsedArgs) int {
				// The configuration (and the .env files) is loaded before
				// the database is created, like initializeContainer upstream.
				if _, ok := r.mustConfig(); !ok {
					return 1
				}
				return exitCode(db.Setup(context.Background(), db.SetupOptions{
					CreateOptions:  db.CreateOptions{DB: a.string("db"), Directory: r.directory, Interactive: a.bool("interactive"), Logger: r.logger},
					MigrateOptions: migrateOptions(a),
					LoadConfig:     r.loadConfig,
				}))
			},
		},
		{
			name: "db:create", desc: "Create the database used by your application",
			options: dbCreateOptions, nodeEnv: "development",
			run: func(r *runner, a parsedArgs) int {
				if _, ok := r.mustConfig(); !ok {
					return 1
				}
				return exitCode(db.Create(context.Background(), db.CreateOptions{DB: a.string("db"), Directory: r.directory, Interactive: a.bool("interactive"), Logger: r.logger}))
			},
		},
		{
			name: "db:migrate", desc: "Migrate the database by executing pending migrations",
			options: append([]option{
				{name: "skip-scripts", kind: boolOption, describe: "Do not run migration scripts"},
				{name: "concurrency", kind: numberOption, describe: "Number of concurrent migrations to run"},
				{name: "all-or-nothing", kind: boolOption, def: false, describe: "If set, the command will fail if any migration fails and revert the migrations that were applied so far"},
			}, migrateLinkSearchOptions...),
			nodeEnv: "development",
			run: func(r *runner, a parsedArgs) int {
				cfg, ok := r.mustConfig()
				if !ok {
					return 1
				}
				return r.report(db.Migrate(context.Background(), cfg, migrateOptions(a)))
			},
		},
		{
			name: "db:migrate:scripts", desc: "Run all migration scripts", nodeEnv: "development",
			run: func(r *runner, a parsedArgs) int {
				_ = os.Setenv("MEDUSA_WORKER_MODE", "server")
				cfg, ok := r.mustConfig()
				if !ok {
					return 1
				}
				return r.report(db.RunMigrationScripts(context.Background(), cfg))
			},
		},
		{
			name: "db:migrate:search", desc: "Create and alter the search indexes declared under search/",
			options: []option{
				{name: "execute-all-search", kind: boolOption, describe: "Skip prompts and execute all (including unsafe) actions on search indexes"},
				{name: "execute-safe-search", kind: boolOption, describe: "Skip prompts and execute only safe actions on search indexes"},
			},
			nodeEnv: "development",
			run: func(r *runner, a parsedArgs) int {
				cfg, ok := r.mustConfig()
				if !ok {
					return 1
				}
				db.MigrateSearch(cfg, r.logger)
				return 0
			},
		},
		{
			name: "db:rollback", usage: "db:rollback [modules...]", desc: "Rollback last batch of executed migrations for a given module",
			nodeEnv: "development", required: 1,
			run: func(r *runner, a parsedArgs) int {
				_ = os.Setenv("MEDUSA_WORKER_MODE", "server")
				cfg, ok := r.mustConfig()
				if !ok {
					return 1
				}
				return r.report(db.Rollback(context.Background(), cfg, a.positional))
			},
		},
		{
			name: "db:generate", usage: "db:generate [modules...]", desc: "Generate migrations for a given module",
			nodeEnv: "development", required: 1,
			run: notSupported("db:generate is not supported by the Go CLI: migrations of Go modules are written as Go code (see packages/core/utils/src/migrations)."),
		},
		{
			name: "plugin:db:generate", desc: "Generate migrations for modules in a plugin", nodeEnv: "development",
			run: notSupported("Plugins are not supported by the Go CLI."),
		},
		{
			name: "db:sync-links", desc: "Sync database schema with the links defined by your application and Medusa core",
			options: []option{
				{name: "execute-all", kind: boolOption, describe: "Skip prompts and execute all (including unsafe) actions"},
				{name: "execute-safe", kind: boolOption, describe: "Skip prompts and execute only safe actions"},
				{name: "concurrency", kind: numberOption, describe: "Number of concurrent migrations to run"},
			},
			nodeEnv: "development",
			run: func(r *runner, a parsedArgs) int {
				cfg, ok := r.mustConfig()
				if !ok {
					return 1
				}
				return r.report(db.SyncLinksCommand(context.Background(), cfg, db.SyncLinksOptions{ExecuteAll: a.bool("execute-all"), ExecuteSafe: a.bool("execute-safe")}))
			},
		},
		{name: "plugin:build", desc: "Build plugin source for publishing to a package registry", nodeEnv: "development", run: notSupported("Plugins are not supported by the Go CLI.")},
		{name: "plugin:develop", desc: "Start plugin development process in watch mode. Changes will be re-published to the local packages registry", nodeEnv: "development", run: notSupported("Plugins are not supported by the Go CLI.")},
		{name: "plugin:publish", desc: "Publish the plugin to the local packages registry", nodeEnv: "development", run: notSupported("Plugins are not supported by the Go CLI.")},
		{name: "plugin:add", usage: "plugin:add [plugin_names...]", desc: "Add the specified plugin to the project from the local packages registry", nodeEnv: "development", required: 1, run: notSupported("Plugins are not supported by the Go CLI.")},
		{
			name: "telemetry", desc: "Enable or disable collection of anonymous usage data.",
			options: []option{
				{name: "enable", kind: boolOption, describe: "Enable telemetry (default)"},
				{name: "disable", kind: boolOption, describe: "Disable telemetry"},
			},
			run: func(r *runner, a parsedArgs) int {
				// The Go CLI collects no telemetry; the setting is only echoed.
				enabled := a.bool("enable") || !a.bool("disable")
				state := "disabled"
				if enabled {
					state = "enabled"
				}
				r.logger.Info(fmt.Sprintf("Telemetry collection %s", state))
				return 0
			},
		},
		{
			name: "codemod", usage: "codemod <codemod-name>", desc: "Run automated code transformations", required: 1,
			options: []option{{name: "dry-run", kind: boolOption, def: false, describe: "Preview changes without modifying files"}},
			run: func(r *runner, a parsedArgs) int {
				// The upstream codemods rewrite TypeScript sources.
				r.logger.Error(fmt.Sprintf("Unknown codemod: %s", a.positional[0]))
				r.logger.Info("\nAvailable codemods:\n")
				return 1
			},
		},
		{
			name: "develop", desc: "Start development server. Watches file and rebuilds when something changes",
			options: append([]option{
				{name: "types", kind: boolOption, def: true, describe: "Generate automated types for modules inside the .medusa directory (no effect)"},
				hostOption, portOption,
			}, lintOptions...),
			nodeEnv: "development",
			run: func(r *runner, a parsedArgs) int {
				if _, ok := r.mustConfig(); !ok {
					return 1
				}
				var startArgs []string
				if a.set["H"] {
					startArgs = append(startArgs, "--host", a.string("H"))
				}
				if a.set["p"] {
					startArgs = append(startArgs, "--port", a.string("p"))
				}
				return r.report(commands.Develop(commands.DevelopOptions{Directory: r.directory, StartArgs: startArgs}))
			},
		},
		{
			name: "start", desc: "Start production server.",
			options: []option{
				{name: "types", kind: boolOption, def: false, describe: "Generate automated types for modules inside the .medusa directory (no effect)"},
				hostOption, portOption,
				{name: "cluster", kind: stringOption, describe: "Start the server in cluster mode (not supported by the Go server)."},
				{name: "workers", kind: stringOption, def: "0", describe: "Number of worker processes in cluster mode (not supported)."},
				{name: "servers", kind: stringOption, def: "0", describe: "Number of server processes in cluster mode (not supported)."},
			},
			nodeEnv: "production",
			run: func(r *runner, a parsedArgs) int {
				cfg, ok := r.mustConfig()
				if !ok {
					return 1
				}
				if a.set["cluster"] {
					r.logger.Warn("Cluster mode is not supported by the Go server: starting a single server process")
				}
				so, ok := r.startOptions(a, cfg)
				if !ok {
					return 1
				}
				return r.report(commands.Start(so))
			},
		},
		{
			name: "build", desc: "Build your project.",
			options: append([]option{
				{name: "admin-only", kind: boolOption, def: false, describe: "Only build the admin to serve it separately (outDir .medusa/admin)"},
				{name: "quiet", kind: boolOption, def: false, describe: "Report lint errors only, suppressing warnings."},
			}, lintOptions...),
			nodeEnv: "development",
			run: func(r *runner, a parsedArgs) int {
				cfg, ok := r.mustConfig()
				if !ok {
					return 1
				}
				return exitCode(commands.Build(commands.BuildOptions{Directory: r.directory, AdminOnly: a.bool("admin-only"), Config: cfg}))
			},
		},
		{
			name: "lint", usage: "lint [paths...]", desc: "Lint your project using its ESLint configuration.",
			options: []option{
				{name: "fix", kind: boolOption, def: false, describe: "Auto-fix lint issues where possible."},
				{name: "quiet", kind: boolOption, def: false, describe: "Report lint errors only, suppressing warnings."},
			},
			nodeEnv: "development",
			run:     notSupported("lint is not supported by the Go CLI: use go vet or your Go linter."),
		},
		{
			name: "user", desc: "Create a user",
			options: []option{
				{name: "e", aliases: []string{"email"}, kind: stringOption, describe: "User's email."},
				{name: "p", aliases: []string{"password"}, kind: stringOption, describe: "User's password."},
				{name: "i", aliases: []string{"id"}, kind: stringOption, describe: "User's id."},
				{name: "invite", kind: boolOption, def: false, describe: "If flag is set, an invitation will be created instead of a new user and the invite token will be returned."},
			},
			run: func(r *runner, a parsedArgs) int {
				cfg, ok := r.mustConfig()
				if !ok {
					return 1
				}
				return r.report(commands.User(commands.UserOptions{
					Email: a.string("e"), Password: a.string("p"), ID: a.string("i"), Invite: a.bool("invite"),
					Config: cfg, Directory: r.directory, OnLoad: r.opts.OnLoad,
				}))
			},
		},
		{
			name: "exec", usage: "exec [file] [args..]", desc: "Run a function defined in a file.", rest: 1,
			run: func(r *runner, a parsedArgs) int {
				cfg, ok := r.mustConfig()
				if !ok {
					return 1
				}
				if len(a.positional) == 0 {
					r.logger.Error("Error running script", "error", errors.New("missing script name"))
					return 1
				}
				return exitCode(commands.Exec(commands.ExecOptions{
					File: a.positional[0], Args: a.positional[1:], Scripts: r.opts.Scripts,
					Config: cfg, Directory: r.directory, OnLoad: r.opts.OnLoad,
				}) == nil)
			},
		},
		{
			name: "mcloud", usage: "mcloud [args..]", desc: "Run mcloud CLI commands through Medusa CLI.", rest: -1,
			run: notSupported("mcloud is not supported by the Go CLI."),
		},
	}
}

// report logs a command error (errors already reported by the command are
// not logged twice) and returns the exit code.
func (r *runner) report(err error) int {
	if err == nil {
		return 0
	}
	if !errors.Is(err, db.ErrDatabaseMissing) {
		r.logger.Error(err.Error())
	}
	return 1
}

func usageLine(c command) string {
	if c.usage != "" {
		return c.usage
	}
	return c.name
}

func showHelp(w io.Writer, cmds []command, c *command) {
	if c != nil {
		fmt.Fprintf(w, "medusa %s\n\n%s\n\nOptions:\n", usageLine(*c), c.desc)
		for _, o := range append(append([]option{}, c.options...), globalOptions...) {
			names := []string{o.name}
			names = append(names, o.aliases...)
			for i, n := range names {
				if len(n) == 1 {
					names[i] = "-" + n
				} else {
					names[i] = "--" + n
				}
			}
			fmt.Fprintf(w, "  %-28s %s\n", strings.Join(names, ", "), o.describe)
		}
		return
	}
	fmt.Fprintln(w, "Usage: medusa <command> [options]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	for _, c := range cmds {
		fmt.Fprintf(w, "  medusa %-28s %s\n", usageLine(c), c.desc)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintf(w, "  %-28s %s\n", "--verbose", "Turn on verbose output")
	fmt.Fprintf(w, "  %-28s %s\n", "--no-color, --no-colors", "Turn off the color in output")
	fmt.Fprintf(w, "  %-28s %s\n", "--json", "Turn on the JSON logger")
	fmt.Fprintf(w, "  %-28s %s\n", "-h, --help", "Show help")
	fmt.Fprintf(w, "  %-28s %s\n", "-v, --version", "Show the version of the Medusa CLI")
}

// didYouMean mirrors did-you-mean.ts (closest command by edit distance).
func didYouMean(arg string, cmds []command) string {
	best, bestDist := "", 4
	for _, c := range cmds {
		if d := levenshtein(arg, c.name); d < bestDist {
			best, bestDist = c.name, d
		}
	}
	if best == "" {
		return ""
	}
	return fmt.Sprintf("Did you mean %q?", best)
}

func levenshtein(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

// Run runs the medusa CLI with args (without the program name) in the
// working directory and returns the process exit code.
func Run(args []string, opts Options) int {
	stdout, stderr := opts.Stdout, opts.Stderr
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	cmds := commandsList()

	directory, err := filepath.Abs(".")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if len(args) == 0 {
		showHelp(stderr, cmds, nil)
		fmt.Fprintln(stderr, "\nPass --help to see all available commands and options.")
		return 1
	}
	switch args[0] {
	case "-h", "--help", "help":
		showHelp(stdout, cmds, nil)
		return 0
	case "-v", "--version", "version":
		fmt.Fprintf(stdout, "Medusa CLI version: %s\nMedusa version: %s\n  Note: this is the Medusa version for the site at: %s\n", Version, Version, directory)
		return 0
	}

	var cmd *command
	for i := range cmds {
		if cmds[i].name == args[0] {
			cmd = &cmds[i]
			break
		}
	}
	if cmd == nil {
		fmt.Fprintf(stderr, "Unknown argument: %s\n\n", args[0])
		if s := didYouMean(args[0], cmds); s != "" {
			fmt.Fprintf(stderr, "%s\n\n", s)
		}
		showHelp(stderr, cmds, nil)
		return 1
	}

	parsed, err := parseArgs(args[1:], cmd.options, cmd.rest)
	if err == nil && len(parsed.positional) < cmd.required {
		err = fmt.Errorf("Not enough non-option arguments: got %d, need at least %d", len(parsed.positional), cmd.required)
	}
	if err != nil {
		fmt.Fprintf(stderr, "%s\n\n", err)
		showHelp(stderr, cmds, cmd)
		return 1
	}
	if parsed.bool("help") {
		showHelp(stdout, cmds, cmd)
		return 0
	}

	// cli.js loads the .env file of the working directory before anything.
	common.ConfigDotenv(filepath.Join(directory, ".env"))
	if cmd.nodeEnv != "" && os.Getenv("NODE_ENV") == "" {
		_ = os.Setenv("NODE_ENV", cmd.nodeEnv)
	}

	logger := reporter.New()
	slog.SetDefault(logger)
	r := &runner{opts: opts, directory: directory, logger: logger}
	return cmd.run(r, parsed)
}
