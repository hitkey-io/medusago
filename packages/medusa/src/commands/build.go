package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	bundler "github.com/hitkey-io/medusago/packages/admin/admin-bundler/src/commands"
	dashboardbundle "github.com/hitkey-io/medusago/packages/admin/dashboard-bundle"
	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/medusa/src/utils"
)

// BuildOptions mirrors the arguments of `medusa build`.
type BuildOptions struct {
	Directory string
	AdminOnly bool
	Config    config.ConfigModule
}

// serverDist mirrors the default tsconfig outDir of compiled applications.
const serverDist = ".medusa/server"

// configFiles are copied next to the compiled server (upstream compiles
// medusa-config.ts into the output directory).
var configFiles = []string{"medusa-config.json", "medusa-config.yaml", "medusa-config.yml"}

func since(t time.Time) string {
	return fmt.Sprintf("%.2f", time.Since(t).Seconds())
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// buildAppBackend mirrors compiler.buildAppBackend: the project is compiled
// into .medusa/server. A project that is a Go module is built with
// `go build`; otherwise the running medusa binary is copied (a project made
// of configuration only). The configuration file is copied along.
func buildAppBackend(dir string, logger *slog.Logger) bool {
	started := time.Now()
	dist := filepath.Join(dir, serverDist)
	logger.Info("Compiling backend source...")
	logger.Info(fmt.Sprintf(`Removing existing "%s" folder`, serverDist))
	if err := os.RemoveAll(dist); err != nil {
		logger.Error(err.Error())
		return false
	}
	if err := os.MkdirAll(dist, 0o755); err != nil {
		logger.Error(err.Error())
		return false
	}

	binary := filepath.Join(dist, "medusa")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		cmd := exec.Command("go", "build", "-o", binary, ".")
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			logger.Warn(fmt.Sprintf("Backend build completed with errors (%ss)", since(started)))
			return false
		}
	} else {
		exe, err := os.Executable()
		if err == nil {
			err = copyFile(exe, binary, 0o755)
		}
		if err != nil {
			logger.Warn("Backend build completed without emitting any output")
			return false
		}
	}
	for _, name := range configFiles {
		src := filepath.Join(dir, name)
		if _, err := os.Stat(src); err == nil {
			if err := copyFile(src, filepath.Join(dist, name), 0o644); err != nil {
				logger.Error(err.Error())
				return false
			}
		}
	}
	logger.Info(fmt.Sprintf("Backend build completed successfully (%ss)", since(started)))
	return true
}

// adminBuildOptions is options.json of build-admin.mjs.
type adminBuildOptions struct {
	ResolveFrom       string   `json:"resolveFrom"`
	WorkDir           string   `json:"workDir"`
	OutDir            string   `json:"outDir"`
	Mode              string   `json:"mode"`
	Path              string   `json:"path"`
	BackendURL        string   `json:"backendUrl"`
	StorefrontURL     string   `json:"storefrontUrl"`
	MaxUploadFileSize *float64 `json:"maxUploadFileSize,omitempty"`
	Sources           []string `json:"sources"`
	Plugins           []string `json:"plugins"`
}

// buildAdminWithNode compiles the dashboard with the project's admin
// extensions through the original @medusajs/admin-bundler (Node).
func buildAdminWithNode(dir, outDir, mode string, admin config.AdminOptions, sources []string) error {
	node, err := exec.LookPath("node")
	if err != nil {
		return errors.New("compiling admin extensions (src/admin) requires Node.js and the @medusajs/admin-bundler package installed in the project")
	}
	work := filepath.Join(dir, ".medusa", "admin-build")
	if err := os.MkdirAll(work, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(work)
	script := filepath.Join(work, "build-admin.mjs")
	if err := os.WriteFile(script, dashboardbundle.BuildScript, 0o644); err != nil {
		return err
	}
	o := adminBuildOptions{
		ResolveFrom:   dir,
		WorkDir:       dir,
		OutDir:        outDir,
		Mode:          mode,
		Path:          admin.Path,
		BackendURL:    admin.BackendURL,
		StorefrontURL: admin.StorefrontURL,
		Sources:       sources,
		Plugins:       []string{},
	}
	if !math.IsInf(admin.MaxUploadFileSize, 0) {
		v := admin.MaxUploadFileSize
		o.MaxUploadFileSize = &v
	}
	raw, _ := json.Marshal(o)
	optionsPath := filepath.Join(work, "options.json")
	if err := os.WriteFile(optionsPath, raw, 0o644); err != nil {
		return err
	}
	cmd := exec.Command(node, script, optionsPath)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// buildAppFrontend mirrors compiler.buildAppFrontend. Without admin
// extensions the dashboard embedded in the binary is written to the output
// directory (no Node needed); with src/admin the dashboard is compiled with
// the original bundler. The server output (.medusa/server/public/admin) is a
// runtime bundle configured from medusa-config when served; the --admin-only
// output (.medusa/admin) has the admin options compiled in, like upstream, so
// it can be hosted separately.
func buildAppFrontend(dir string, adminOnly bool, cfg config.ConfigModule, logger *slog.Logger) bool {
	started := time.Now()
	admin := cfg.Admin
	if admin.Disable && !adminOnly {
		logger.Info("Skipping admin build, since its disabled inside the medusa-config file")
		return true
	}
	if !admin.Disable && adminOnly {
		logger.Warn("You are building using the flag --admin-only but the admin is enabled in your medusa-config, If you intend to host the dashboard separately you should disable the admin in your medusa config")
	}

	outDir := filepath.Join(dir, serverDist, utils.ADMIN_RELATIVE_OUTPUT_DIR)
	if adminOnly {
		outDir = filepath.Join(dir, utils.ADMIN_ONLY_OUTPUT_DIR)
	}

	logger.Info("Compiling frontend source...")
	var err error
	sourceDir := filepath.Join(dir, utils.ADMIN_SOURCE_DIR)
	if info, statErr := os.Stat(sourceDir); statErr == nil && info.IsDir() {
		mode := "runtime"
		if adminOnly {
			mode = "static"
		}
		err = buildAdminWithNode(dir, outDir, mode, admin, []string{sourceDir})
	} else if adminOnly {
		var b *bundler.Bundle
		b, err = bundler.LoadBundle(dashboardbundle.App(), bundler.BundlerOptions{
			Path:              admin.Path,
			BackendURL:        admin.BackendURL,
			StorefrontURL:     admin.StorefrontURL,
			MaxUploadFileSize: admin.MaxUploadFileSize,
		})
		if err == nil {
			err = b.WriteTo(outDir)
		}
	} else {
		err = bundler.CopyFS(dashboardbundle.App(), outDir)
	}
	if err != nil {
		logger.Error("Unable to compile frontend source")
		logger.Error(err.Error())
		return false
	}
	logger.Info(fmt.Sprintf("Frontend build completed successfully (%ss)", since(started)))
	return true
}

// Build mirrors build.ts.
func Build(opts BuildOptions) bool {
	logger := slog.Default()
	logger.Info("Starting build...")
	ok := true
	if !opts.AdminOnly {
		ok = buildAppBackend(opts.Directory, logger)
	}
	if !buildAppFrontend(opts.Directory, opts.AdminOnly, opts.Config, logger) {
		ok = false
	}
	return ok
}
