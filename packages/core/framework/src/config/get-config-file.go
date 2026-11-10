package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// GetConfigFile mirrors getConfigFile(rootDir, configName) for the stock
// binary: it reads <rootDir>/<configName>.json (the JSON form of the object
// passed to defineConfig in medusa-config.js). A missing file yields an empty
// input config and an empty path, so the defineConfig defaults apply.
func GetConfigFile(rootDir, configName string) (InputConfig, string, error) {
	var input InputConfig
	path := filepath.Join(rootDir, configName+".json")
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return input, "", nil
	}
	if err != nil {
		return input, path, err
	}
	if err := json.Unmarshal(raw, &input); err != nil {
		return input, path, err
	}
	return input, path, nil
}
