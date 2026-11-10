package main

import (
	"context"
	"os"
	"path/filepath"
)

const (
	defaultRepo = "https://github.com/hitkey-io/medusago-starter"
	branch      = "main"
)

// runCloneRepo mirrors runCloneRepo: a shallow clone of the starter whose
// .git directory is removed.
func runCloneRepo(ctx context.Context, projectPath, repoURL string, verbose bool) error {
	if repoURL == "" {
		repoURL = defaultRepo
	}
	if _, err := execute(ctx, "", nil, verbose, "git", "clone", repoURL, "-b", branch, projectPath, "--depth", "1"); err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(projectPath, ".git"))
}
