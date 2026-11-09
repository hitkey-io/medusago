package common

import (
	"os"

	modulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// HttpConfig mirrors projectConfig.http.
type HttpConfig struct {
	StoreCors    string `json:"storeCors" yaml:"storeCors"`
	AdminCors    string `json:"adminCors" yaml:"adminCors"`
	AuthCors     string `json:"authCors" yaml:"authCors"`
	JwtSecret    string `json:"jwtSecret" yaml:"jwtSecret"`
	CookieSecret string `json:"cookieSecret" yaml:"cookieSecret"`
}

// ProjectConfig mirrors projectConfig from medusa-config.
type ProjectConfig struct {
	DatabaseURL string     `json:"databaseUrl" yaml:"databaseUrl"`
	RedisURL    string     `json:"redisUrl" yaml:"redisUrl"`
	HTTP        HttpConfig `json:"http" yaml:"http"`
}

// ModuleProviderConfig is a provider registered on a module.
type ModuleProviderConfig struct {
	Resolve string         `json:"resolve" yaml:"resolve"`
	ID      string         `json:"id" yaml:"id"`
	Options map[string]any `json:"options,omitempty" yaml:"options,omitempty"`
}

// ModuleConfig is one entry of the resolved modules list.
type ModuleConfig struct {
	Key       string                 `json:"key" yaml:"key"`
	Resolve   string                 `json:"resolve" yaml:"resolve"`
	Disable   bool                   `json:"disable,omitempty" yaml:"disable,omitempty"`
	Providers []ModuleProviderConfig `json:"providers,omitempty" yaml:"providers,omitempty"`
}

// InputConfig is what a project passes to DefineConfig.
type InputConfig struct {
	ProjectConfig ProjectConfig  `json:"projectConfig" yaml:"projectConfig"`
	Modules       []ModuleConfig `json:"modules,omitempty" yaml:"modules,omitempty"`
}

// ConfigModule is the resolved configuration.
type ConfigModule struct {
	ProjectConfig ProjectConfig
	Modules       []ModuleConfig
}

// DefineConfig mirrors defineConfig from define-config.ts: it fills project
// defaults and merges the user modules over the default modules, keeping the
// upstream order.
func DefineConfig(input InputConfig) ConfigModule {
	pc := input.ProjectConfig
	if pc.DatabaseURL == "" {
		pc.DatabaseURL = os.Getenv("DATABASE_URL")
	}
	if pc.RedisURL == "" {
		pc.RedisURL = os.Getenv("REDIS_URL")
	}
	if pc.HTTP.JwtSecret == "" {
		pc.HTTP.JwtSecret = envOr("JWT_SECRET", "supersecret")
	}
	if pc.HTTP.CookieSecret == "" {
		pc.HTTP.CookieSecret = envOr("COOKIE_SECRET", "supersecret")
	}
	if pc.HTTP.StoreCors == "" {
		pc.HTTP.StoreCors = os.Getenv("STORE_CORS")
	}
	if pc.HTTP.AdminCors == "" {
		pc.HTTP.AdminCors = os.Getenv("ADMIN_CORS")
	}
	if pc.HTTP.AuthCors == "" {
		pc.HTTP.AuthCors = os.Getenv("AUTH_CORS")
	}

	modules := defaultModules()
	for _, um := range input.Modules {
		replaced := false
		for i := range modules {
			if modules[i].Key == um.Key {
				modules[i] = um
				replaced = true
				break
			}
		}
		if !replaced {
			modules = append(modules, um)
		}
	}

	return ConfigModule{ProjectConfig: pc, Modules: modules}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mod(key string, providers ...ModuleProviderConfig) ModuleConfig {
	return ModuleConfig{Key: key, Resolve: modulessdk.MODULE_PACKAGE_NAMES[key], Providers: providers}
}

// defaultModules mirrors the self-hosted default modules of define-config.ts.
func defaultModules() []ModuleConfig {
	return []ModuleConfig{
		mod(modulessdk.STOCK_LOCATION),
		mod(modulessdk.INVENTORY),
		mod(modulessdk.PRODUCT),
		mod(modulessdk.PRICING),
		mod(modulessdk.PROMOTION),
		mod(modulessdk.CUSTOMER),
		mod(modulessdk.SALES_CHANNEL),
		mod(modulessdk.CART),
		mod(modulessdk.REGION),
		mod(modulessdk.API_KEY),
		mod(modulessdk.STORE),
		mod(modulessdk.TAX),
		mod(modulessdk.CURRENCY),
		mod(modulessdk.PAYMENT),
		mod(modulessdk.ORDER),
		mod(modulessdk.SETTINGS),
		{
			Key:     modulessdk.TRANSLATION,
			Resolve: modulessdk.MODULE_PACKAGE_NAMES[modulessdk.TRANSLATION],
			Disable: true, // enabled by the "translation" feature flag upstream
		},
		mod(modulessdk.AUTH, ModuleProviderConfig{Resolve: "@medusajs/medusa/auth-emailpass", ID: "emailpass"}),
		mod(modulessdk.USER),
		mod(modulessdk.FULFILLMENT, ModuleProviderConfig{Resolve: "@medusajs/medusa/fulfillment-manual", ID: "manual"}),
		mod(modulessdk.NOTIFICATION, ModuleProviderConfig{
			Resolve: "@medusajs/medusa/notification-local",
			ID:      "local",
			Options: map[string]any{"name": "Local Notification Provider", "channels": []string{"feed"}},
		}),
		mod(modulessdk.SEARCH, ModuleProviderConfig{
			Resolve: "@medusajs/medusa/search-postgres",
			ID:      "postgres",
			Options: map[string]any{"engine": "native"},
		}),
		mod(modulessdk.CACHE),
		mod(modulessdk.EVENT_BUS),
		mod(modulessdk.WORKFLOW_ENGINE),
		mod(modulessdk.LOCKING),
		mod(modulessdk.FILE, ModuleProviderConfig{
			Resolve: "@medusajs/medusa/file-local",
			ID:      "local",
		}),
	}
}
