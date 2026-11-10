package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// startMedusa mirrors start-medusa.ts: the development server of the
// backend (`go run . develop`, the equivalent of `npm run dev`).
func startMedusa(ctx context.Context, backend string) {
	startProcess(ctx, backend, "go", "run", ".", "develop")
}

func startProcess(ctx context.Context, dir, name string, args ...string) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		logMessage(fmt.Sprintf("Couldn't start %s: %v", name, err), "warn")
		return
	}
	go func() { _ = cmd.Wait() }()
}

// waitOn mirrors wait-on for one HTTP resource.
func waitOn(ctx context.Context, url string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if res, err := client.Get(url); err == nil {
			res.Body.Close()
			if res.StatusCode < 400 {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

// openURL mirrors open().
func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

// execute mirrors execute.ts: the command output is returned (and shown
// with --verbose).
func execute(ctx context.Context, dir string, env []string, verbose bool, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var buf bytes.Buffer
	if verbose {
		cmd.Stdout = io.MultiWriter(&buf, os.Stdout)
		cmd.Stderr = io.MultiWriter(&buf, os.Stderr)
	} else {
		cmd.Stdout = &buf
		cmd.Stderr = &buf
	}
	err := cmd.Run()
	if err != nil && !verbose {
		return buf.String(), fmt.Errorf("%s %v: %v\n%s", name, args, err, buf.String())
	}
	return buf.String(), err
}
