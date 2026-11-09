// Package commands mirrors @medusajs/medusa/src/commands.
package commands

import (
	"context"
	"errors"
	"fmt"
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
	Host string
	Port int
}

// NewApp builds the HTTP handler served by `medusa start`.
func NewApp() http.Handler {
	mux := http.NewServeMux()

	// Health check endpoint, registered the same way as in start.ts.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	return mux
}

// Start runs the HTTP server until the process receives SIGINT/SIGTERM.
func Start(opts StartOptions) error {
	if opts.Port == 0 {
		opts.Port = 9000
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := &http.Server{
		Addr:              net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port)),
		Handler:           NewApp(),
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
