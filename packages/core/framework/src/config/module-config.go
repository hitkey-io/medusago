package config

import (
	"encoding/json"
	"strings"

	modulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// ModuleKeyFromResolve returns the key of a Medusa module from its package
// name ("@medusajs/medusa/<name>" or "@medusajs/<name>"), like upstream
// derives it from the module definition; "" for other packages.
func ModuleKeyFromResolve(resolve string) string {
	name := strings.TrimPrefix(strings.TrimPrefix(resolve, "@medusajs/medusa/"), "@medusajs/")
	for key, pkg := range modulessdk.MODULE_PACKAGE_NAMES {
		if pkg == resolve || strings.TrimPrefix(pkg, "@medusajs/medusa/") == name {
			return key
		}
	}
	return ""
}

// UnmarshalJSON accepts the medusa-config module format as well:
// { resolve, key?, disable?, options: { providers } } (the providers of the
// module options become Providers).
func (m *ModuleConfig) UnmarshalJSON(data []byte) error {
	type plain ModuleConfig
	var in struct {
		plain
		Options *struct {
			Providers []ModuleProviderConfig `json:"providers"`
		} `json:"options"`
	}
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	*m = ModuleConfig(in.plain)
	if len(m.Providers) == 0 && in.Options != nil {
		m.Providers = in.Options.Providers
	}
	return nil
}
