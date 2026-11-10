package main

import "path/filepath"

func getBackendDirectory(projectDirectory string) string {
	return filepath.Join(projectDirectory, "apps", "backend")
}

func getStorefrontDirectory(projectDirectory string) string {
	return filepath.Join(projectDirectory, "apps", "storefront")
}
