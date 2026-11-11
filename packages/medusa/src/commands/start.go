// Package commands mirrors @medusajs/medusa/src/commands.
package commands

import (
	"context"
	"errors"
	"fmt"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
)

// StartOptions mirrors the options accepted by `medusa start`.
type StartOptions struct {
	Host   string
	Port   int
	Config config.ConfigModule
	// Directory is the project root (defaults to the working directory).
	Directory string
	// OnLoad hooks run once the application is loaded (see loaders.Options).
	OnLoad []func(ctx context.Context, app *loaders.MedusaApp) error
	// Routes and Middlewares are the project API routes (see loaders.Options).
	Routes      []medusahttp.RouteFile
	Middlewares []medusahttp.MiddlewareRoute
}

// displayAdminURL mirrors displayAdminUrl: outside production, the admin
// URL is logged when the admin is enabled.
func displayAdminURL(logger *slog.Logger, host string, port int, cfg config.ConfigModule) {
	if config.IsProduction() || cfg.Admin.Disable {
		return
	}
	if host == "" {
		host = "localhost"
	}
	logger.Info(fmt.Sprintf("Admin URL → http://%s:%d%s", host, port, cfg.Admin.Path))
}

// Start runs the HTTP server until the process receives SIGINT/SIGTERM.
func Start(opts StartOptions) error {
	if opts.Port == 0 {
		opts.Port = 9000
	}
	logger := slog.Default()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("Creating server")
	app, err := loaders.InitializeWithOptions(ctx, opts.Config, loaders.Options{Directory: opts.Directory, OnLoad: opts.OnLoad, Routes: opts.Routes, Middlewares: opts.Middlewares})
	if err != nil {
		logger.Error("Error starting server", "error", err)
		return err
	}
	defer app.Shutdown()

	listener, err := net.Listen("tcp", net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port)))
	if err != nil {
		logger.Error("Error starting server", "error", err)
		return err
	}
	server := &http.Server{
		Handler:           app.Handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()
	logger.Info(fmt.Sprintf("Server is ready on port: %d", opts.Port))
	displayAdminURL(logger, opts.Host, opts.Port, opts.Config)

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		logger.Info("Gracefully shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("Error received when shutting down the server.", "error", err)
			return err
		}
		return nil
	}
}
