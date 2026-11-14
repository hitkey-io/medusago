package main

// region: config
import (
	"os"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
)

// medusaConfig is the equivalent of medusa-config.ts. The CLI loads the .env
// files before calling it.
func medusaConfig() config.InputConfig {
	return config.InputConfig{
		ProjectConfig: config.ProjectConfig{
			DatabaseURL: os.Getenv("DATABASE_URL"),
			HTTP: config.HttpConfig{
				StoreCors:    os.Getenv("STORE_CORS"),
				AdminCors:    os.Getenv("ADMIN_CORS"),
				AuthCors:     os.Getenv("AUTH_CORS"),
				JwtSecret:    os.Getenv("JWT_SECRET"),
				CookieSecret: os.Getenv("COOKIE_SECRET"),
			},
		},
	}
}

// endregion: config
