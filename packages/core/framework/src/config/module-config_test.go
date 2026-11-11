package config

import (
	"encoding/json"
	"testing"
)

func TestModuleConfigFromMedusaConfigFormat(t *testing.T) {
	var in InputConfig
	raw := `{"projectConfig":{},"modules":[
		{"resolve":"@medusajs/medusa/file","options":{"providers":[{"resolve":"@medusajs/medusa/file-s3","id":"s3","options":{"bucket":"b"}}]}},
		{"resolve":"@medusajs/analytics","options":{"providers":[{"resolve":"@medusajs/medusa/analytics-local","id":"local"}]}},
		{"key":"custom","resolve":"./modules/custom"}
	]}`
	if err := json.Unmarshal([]byte(raw), &in); err != nil {
		t.Fatal(err)
	}
	cfg := DefineConfig(in)
	byKey := map[string]ModuleConfig{}
	for _, m := range cfg.Modules {
		byKey[m.Key] = m
	}
	file := byKey["file"]
	if file.Resolve != "@medusajs/medusa/file" || len(file.Providers) != 1 || file.Providers[0].Resolve != "@medusajs/medusa/file-s3" || file.Providers[0].Options["bucket"] != "b" {
		t.Errorf("file module: %+v", file)
	}
	if a := byKey["analytics"]; len(a.Providers) != 1 || a.Providers[0].ID != "local" {
		t.Errorf("analytics module: %+v", a)
	}
	if _, ok := byKey["custom"]; !ok {
		t.Error("custom module missing")
	}
	n := 0
	for _, m := range cfg.Modules {
		if m.Key == "file" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("file module listed %d times", n)
	}
}
