package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// checkRequirements replaces validateNodeVersion: the backend is a Go
// module, so Go and Git are required.
func checkRequirements(opts ProjectOptions) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("MedusaGo requires Git. Please install Git and try again.")
	}
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("MedusaGo requires Go 1.26 or later. Please install Go (https://go.dev/dl/) and try again.")
	}
	return nil
}

// checkNodeRequirements: the Next.js storefront is a Node.js application.
func checkNodeRequirements() error {
	if _, err := exec.LookPath("node"); err != nil {
		return fmt.Errorf("The storefront requires Node.js. Please install Node.js (https://nodejs.org/en/download) and try again.")
	}
	return nil
}

func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

// getProjectName mirrors ProjectCreatorFactory.getProjectName.
func getProjectName(args []string, directoryPath string) (string, error) {
	ask := len(args) == 0
	if len(args) > 0 {
		if isDir(filepath.Join(directoryPath, args[0])) {
			logMessage(fmt.Sprintf("A directory already exists with the name %s. Please enter a different project name.", args[0]), "warn")
			ask = true
		} else if strings.Contains(args[0], ".") {
			logMessage("Project names cannot contain a dot (.) character. Please enter a different project name.", "warn")
			ask = true
		}
	}
	if !ask {
		return args[0], nil
	}
	for {
		name := strings.ToLower(slugify(promptInput("What's the name of your project?", "my-medusa-store")))
		switch {
		case strings.Contains(name, "."):
			fmt.Println(">> Project names cannot contain a dot (.) character. Please enter a different project name.")
		case name == "":
			fmt.Println(">> Please enter a project name")
		case isDir(filepath.Join(directoryPath, name)):
			fmt.Println(">> A directory already exists with the same name. Please enter a different project name.")
		default:
			return name, nil
		}
	}
}
