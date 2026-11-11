package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// configExtensions are the forms of the configuration file of the stock
// binary, in lookup order.
var configExtensions = []string{".json", ".yaml", ".yml"}

// GetConfigFile mirrors getConfigFile(rootDir, configName) for the stock
// binary: it reads <rootDir>/<configName>.json, .yaml or .yml (the first that
// exists), the JSON or YAML form of the object passed to defineConfig in
// medusa-config.js. A missing file yields an empty input config and an empty
// path, so the defineConfig defaults apply.
func GetConfigFile(rootDir, configName string) (InputConfig, string, error) {
	var input InputConfig
	for _, ext := range configExtensions {
		path := filepath.Join(rootDir, configName+ext)
		raw, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return input, path, err
		}
		if ext != ".json" {
			if raw, err = yamlToJSON(raw); err != nil {
				return input, path, err
			}
		}
		if err := json.Unmarshal(raw, &input); err != nil {
			return input, path, err
		}
		return input, path, nil
	}
	return input, "", nil
}

// yamlToJSON converts a YAML document to JSON, so the YAML form decodes with
// the JSON field names of the configuration.
func yamlToJSON(raw []byte) ([]byte, error) {
	var doc any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	normalized, err := jsonCompatible(doc)
	if err != nil {
		return nil, err
	}
	if normalized == nil {
		normalized = map[string]any{}
	}
	return json.Marshal(normalized)
}

func jsonCompatible(v any) (any, error) {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, x := range t {
			c, err := jsonCompatible(x)
			if err != nil {
				return nil, err
			}
			out[k] = c
		}
		return out, nil
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, x := range t {
			c, err := jsonCompatible(x)
			if err != nil {
				return nil, err
			}
			out[fmt.Sprint(k)] = c
		}
		return out, nil
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			c, err := jsonCompatible(x)
			if err != nil {
				return nil, err
			}
			out[i] = c
		}
		return out, nil
	}
	return v, nil
}
