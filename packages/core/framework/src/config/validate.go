package config

import (
	"fmt"
	"log/slog"
)

// Validate mirrors the http part of ConfigManager.buildHttpConfig: a missing
// http.jwtSecret or http.cookieSecret is an error in production and a warning
// elsewhere (where defineConfig has already filled the default, so the
// warning only shows for configurations built without it).
func Validate(cfg ConfigModule, logger *slog.Logger) error {
	http := cfg.ProjectConfig.HTTP
	for _, m := range []struct{ name, value string }{
		{"http.jwtSecret", http.JwtSecret},
		{"http.cookieSecret", http.CookieSecret},
	} {
		if m.value != "" {
			continue
		}
		if IsProduction() {
			return fmt.Errorf("[config] ⚠️ %s not found.", m.name)
		}
		if logger != nil {
			logger.Warn(m.name + " not found. Using default 'supersecret'.")
		}
	}
	return nil
}
