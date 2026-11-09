// Package commands mirrors @medusajs/medusa/src/commands.
package commands

import (
	"context"
	"errors"
	"fmt"
	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// StartOptions mirrors the options accepted by `medusa start`.
type StartOptions struct {
	Host   string
	Port   int
	Config config.ConfigModule
}

// Start runs the HTTP server until the process receives SIGINT/SIGTERM.
func Start(opts StartOptions) error {
	if opts.Port == 0 {
		opts.Port = 9000
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := loaders.Initialize(ctx, opts.Config)
	if err != nil {
		return err
	}
	defer app.Shutdown()

	server := &http.Server{
		Addr:              net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port)),
		Handler:           app.Handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info(fmt.Sprintf("Server is ready on port: %d", opts.Port))
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}
