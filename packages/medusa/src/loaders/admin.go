package loaders

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"

	bundler "github.com/hitkey-io/medusago/packages/admin/admin-bundler/src/commands"
	dashboardbundle "github.com/hitkey-io/medusago/packages/admin/dashboard-bundle"
	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/utils"
)

// notAllowedAdminPaths mirrors NOT_ALLOWED_PATHS of loaders/admin.ts.
var notAllowedAdminPaths = []string{"/auth", "/store", "/admin"}

// AdminLoader mirrors adminLoader: unless admin.disable, the dashboard is
// served at admin.path.
//
// Upstream serves <root>/public/admin (the output of `medusa build`) and, in
// development, the Vite dev server. Here the build directory is served when it
// exists (a build with admin extensions); otherwise, in every environment, the
// dashboard bundle embedded in the binary.
func AdminLoader(app *medusahttp.App, cfg config.ConfigModule, rootDirectory string, logger *slog.Logger) error {
	admin := cfg.Admin
	if admin.Disable {
		return nil
	}
	if slices.Contains(notAllowedAdminPaths, admin.Path) {
		logger.Error(fmt.Sprintf("The 'admin.path' in 'medusa-config.js' is set to a value that is not allowed. This can prevent your server from working correctly. Please set 'admin.path' to a value that is not one of the following: %s.", strings.Join(notAllowedAdminPaths, ", ")))
	}
	// the dashboard of the configured plugins (an optional plugin with an
	// admin extension has its own build)
	var names []string
	for _, p := range cfg.Plugins {
		names = append(names, p.Resolve)
	}
	fallback, err := dashboardbundle.AppFor(names)
	if err != nil {
		return err
	}
	mw, err := bundler.Serve(bundler.ServeOptions{
		OutDir:   filepath.Join(rootDirectory, utils.ADMIN_RELATIVE_OUTPUT_DIR),
		Fallback: fallback,
		BundlerOptions: bundler.BundlerOptions{
			Path:              admin.Path,
			BackendURL:        admin.BackendURL,
			StorefrontURL:     admin.StorefrontURL,
			MaxUploadFileSize: admin.MaxUploadFileSize,
		},
	})
	if err != nil {
		return err
	}
	app.Use(admin.Path, mw)
	return nil
}
