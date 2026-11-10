package main

import (
	"context"
	"os"
	"path/filepath"
)

// askForNextjsStarter mirrors askForNextjsStarter.
func askForNextjsStarter() bool {
	return promptConfirm("Would you like to install the storefront? You can also install it later.", false)
}

// installNextjsStarter mirrors installNextjsStarter.
func installNextjsStarter(storefront string, s *spinner) error {
	s.title("Setting up storefront...")
	template := filepath.Join(storefront, ".env.template")
	if _, err := os.Stat(template); err == nil {
		if err := os.Rename(template, filepath.Join(storefront, ".env.local")); err != nil {
			return err
		}
	}
	s.fact("Storefront is set up in the " + storefront + " directory.")
	return nil
}

func storefrontPM(pm string) string {
	if pm == "" {
		return "npm"
	}
	return pm
}

func storefrontCommand(pm, script string) string {
	pm = storefrontPM(pm)
	if pm == "npm" {
		return "npm run " + script
	}
	return pm + " " + script
}

// startNextjsStarter mirrors startNextjsStarter.
func startNextjsStarter(ctx context.Context, dir, pm string) {
	pm = storefrontPM(pm)
	args := []string{"run", "dev"}
	if pm != "npm" {
		args = []string{"dev"}
	}
	startProcess(ctx, dir, pm, args...)
}
