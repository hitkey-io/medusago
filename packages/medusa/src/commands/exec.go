package commands

import (
	"context"
	"fmt"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	typescommon "github.com/hitkey-io/medusago/packages/core/types/src/common"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
)

// ExecOptions mirrors the arguments of `medusa exec [file] [args..]`.
type ExecOptions struct {
	// File names the script: a registered name ("seed") or the path of its
	// source file ("./src/scripts/seed.go", "src/scripts/seed.ts").
	File string
	Args []string
	// Scripts are the scripts registered by the project, by name.
	Scripts   map[string]typescommon.ExecScript
	Config    config.ConfigModule
	Directory string
	OnLoad    []func(ctx context.Context, app *loaders.MedusaApp) error
	// Routes and Middlewares are the project API routes (see loaders.Options).
	Routes      []medusahttp.RouteFile
	Middlewares []medusahttp.MiddlewareRoute
}

// scriptExtensions are dropped when a script is named by its file path.
var scriptExtensions = []string{".go", ".ts", ".js", ".mjs", ".cjs", ".mts", ".cts"}

// ResolveScript finds the script named by file. Go cannot import a file at
// run time like upstream's dynamicImport, so scripts are registered by name
// and a file path is mapped to a name: "./src/scripts/seed.ts" tries
// "./src/scripts/seed.ts", "src/scripts/seed.ts", "src/scripts/seed", then
// "seed" (the path relative to src/scripts, without extension).
func ResolveScript(scripts map[string]typescommon.ExecScript, file string) (string, typescommon.ExecScript, bool) {
	candidates := []string{file}
	p := path.Clean(filepath.ToSlash(file))
	candidates = append(candidates, p)
	for _, ext := range scriptExtensions {
		if strings.HasSuffix(p, ext) {
			p = strings.TrimSuffix(p, ext)
			candidates = append(candidates, p)
			break
		}
	}
	if rel, ok := strings.CutPrefix(p, "src/scripts/"); ok {
		candidates = append(candidates, rel)
	}
	if rel, ok := strings.CutPrefix(p, "scripts/"); ok {
		candidates = append(candidates, rel)
	}
	for _, c := range candidates {
		if s, ok := scripts[c]; ok && s != nil {
			return c, s, true
		}
	}
	return "", nil, false
}

// Exec mirrors exec.ts: the script receives the container of the loaded
// application (without the admin) and the remaining arguments.
func Exec(opts ExecOptions) error {
	logger := slog.Default()
	_ = os.Setenv("MEDUSA_WORKER_MODE", "server")
	logger.Info(fmt.Sprintf("Executing script at %s...", opts.File))

	_, script, ok := ResolveScript(opts.Scripts, opts.File)
	if !ok {
		names := make([]string, 0, len(opts.Scripts))
		for n := range opts.Scripts {
			names = append(names, n)
		}
		sort.Strings(names)
		available := "none"
		if len(names) > 0 {
			available = strings.Join(names, ", ")
		}
		err := fmt.Errorf("Script %s is not registered. Registered scripts: %s", opts.File, available)
		logger.Error("Error running script", "error", err)
		return err
	}

	ctx := context.Background()
	app, err := loaders.InitializeWithOptions(ctx, opts.Config, loaders.Options{Directory: opts.Directory, SkipAdmin: true, OnLoad: opts.OnLoad, Routes: opts.Routes, Middlewares: opts.Middlewares})
	if err != nil {
		logger.Error("Error running script", "error", err)
		return err
	}
	defer app.Shutdown()

	if err := script(ctx, typescommon.ExecArgs{Container: app.Container, Args: opts.Args}); err != nil {
		logger.Error("Error running script", "error", err)
		return err
	}
	logger.Info("Finished executing script.")
	return nil
}
