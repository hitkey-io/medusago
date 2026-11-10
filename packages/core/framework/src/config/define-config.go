// Package config mirrors @medusajs/framework/src/config.
package config

import (
	"os"

	ff "github.com/hitkey-io/medusago/packages/core/utils/src/feature-flags"
	modulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// HttpConfig mirrors projectConfig.http.
type HttpConfig struct {
	StoreCors    string `json:"storeCors" yaml:"storeCors"`
	AdminCors    string `json:"adminCors" yaml:"adminCors"`
	AuthCors     string `json:"authCors" yaml:"authCors"`
	JwtSecret    string `json:"jwtSecret" yaml:"jwtSecret"`
	CookieSecret string `json:"cookieSecret" yaml:"cookieSecret"`
	// JwtExpiresIn is a jsonwebtoken/ms duration ("1d", "10h", "60s") or a
	// number of seconds.
	JwtExpiresIn        string `json:"jwtExpiresIn" yaml:"jwtExpiresIn"`
	StoreRelationsLimit int    `json:"storeRelationsLimit" yaml:"storeRelationsLimit"`
	// AuthMethodsPerActor lists the auth providers allowed per actor type.
	AuthMethodsPerActor map[string][]string `json:"authMethodsPerActor" yaml:"authMethodsPerActor"`
	// RequireActorAssociation, per auth provider.
	RequireActorAssociation map[string]bool `json:"requireActorAssociation" yaml:"requireActorAssociation"`
	RestrictedFields        struct {
		Store []string `json:"store" yaml:"store"`
	} `json:"restrictedFields" yaml:"restrictedFields"`
}

// SessionOptions mirrors projectConfig.sessionOptions.
type SessionOptions struct {
	Name              string `json:"name" yaml:"name"`
	Resave            *bool  `json:"resave" yaml:"resave"`
	Rolling           *bool  `json:"rolling" yaml:"rolling"`
	SaveUninitialized *bool  `json:"saveUninitialized" yaml:"saveUninitialized"`
	Secret            string `json:"secret" yaml:"secret"`
	// Ttl is the session cookie maxAge in milliseconds.
	Ttl *int64 `json:"ttl" yaml:"ttl"`
}

// CookieOptions mirrors projectConfig.cookieOptions (express-session cookie
// options, spread over the defaults).
type CookieOptions struct {
	Secure   *bool  `json:"secure" yaml:"secure"`
	SameSite any    `json:"sameSite" yaml:"sameSite"`
	Domain   string `json:"domain" yaml:"domain"`
	Path     string `json:"path" yaml:"path"`
	HttpOnly *bool  `json:"httpOnly" yaml:"httpOnly"`
	MaxAge   *int64 `json:"maxAge" yaml:"maxAge"`
	// Partitioned and Priority are passed through to cookie.serialize.
	Partitioned *bool  `json:"partitioned" yaml:"partitioned"`
	Priority    string `json:"priority" yaml:"priority"`
}

// ProjectConfig mirrors projectConfig from medusa-config.
type ProjectConfig struct {
	DatabaseURL string     `json:"databaseUrl" yaml:"databaseUrl"`
	RedisURL    string     `json:"redisUrl" yaml:"redisUrl"`
	HTTP        HttpConfig `json:"http" yaml:"http"`
	// RedisPrefix prefixes the Redis keys (sessions: <prefix>sess:<sid>).
	RedisPrefix string `json:"redisPrefix" yaml:"redisPrefix"`
	// RedisOptions are the ioredis options of the session store client
	// (merged under the redisUrl values like `new Redis(redisUrl, redisOptions)`).
	RedisOptions   map[string]any  `json:"redisOptions" yaml:"redisOptions"`
	SessionOptions *SessionOptions `json:"sessionOptions" yaml:"sessionOptions"`
	CookieOptions  *CookieOptions  `json:"cookieOptions" yaml:"cookieOptions"`
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
	// FeatureFlags mirrors config.featureFlags.
	FeatureFlags map[string]any `json:"featureFlags,omitempty" yaml:"featureFlags,omitempty"`
	// Plugins mirrors config.plugins.
	Plugins []PluginConfig `json:"plugins,omitempty" yaml:"plugins,omitempty"`
}

// ConfigModule is the resolved configuration.
type ConfigModule struct {
	ProjectConfig ProjectConfig
	Modules       []ModuleConfig
	FeatureFlags  map[string]any
	Plugins       []PluginConfig
}

const medusaCloudExecutionContext = "medusa-cloud"

// DefineConfig mirrors defineConfig from define-config.ts: it fills project
// defaults and merges the user modules over the default modules, keeping the
// upstream order.
func DefineConfig(input InputConfig) ConfigModule {
	pc := input.ProjectConfig
	if pc.DatabaseURL == "" {
		pc.DatabaseURL = os.Getenv("DATABASE_URL")
	}
	// REDIS_URL is only a projectConfig default on Medusa Cloud.
	if pc.RedisURL == "" && os.Getenv("EXECUTION_CONTEXT") == medusaCloudExecutionContext {
		pc.RedisURL = os.Getenv("REDIS_URL")
	}
	if pc.HTTP.JwtSecret == "" {
		pc.HTTP.JwtSecret = envOr("JWT_SECRET", "supersecret")
	}
	if pc.HTTP.CookieSecret == "" {
		pc.HTTP.CookieSecret = envOr("COOKIE_SECRET", "supersecret")
	}
	if pc.HTTP.StoreCors == "" {
		pc.HTTP.StoreCors = envOr("STORE_CORS", defaultStoreCors)
	}
	if pc.HTTP.AdminCors == "" {
		pc.HTTP.AdminCors = envOr("ADMIN_CORS", defaultAdminCors)
	}
	if pc.HTTP.AuthCors == "" {
		pc.HTTP.AuthCors = envOr("AUTH_CORS", defaultAdminCors)
	}
	if pc.HTTP.JwtExpiresIn == "" {
		pc.HTTP.JwtExpiresIn = "1d"
	}
	if pc.HTTP.StoreRelationsLimit == 0 {
		pc.HTTP.StoreRelationsLimit = 3
	}
	if pc.HTTP.RestrictedFields.Store == nil {
		pc.HTTP.RestrictedFields.Store = DefaultStoreRestrictedFields
	}
	if pc.HTTP.AuthMethodsPerActor == nil {
		pc.HTTP.AuthMethodsPerActor = map[string][]string{}
	}
	if pc.HTTP.AuthMethodsPerActor["user"] == nil {
		pc.HTTP.AuthMethodsPerActor["user"] = []string{"emailpass"}
	}

	modules := defaultModules(input.FeatureFlags)
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

	return ConfigModule{ProjectConfig: pc, Modules: modules, FeatureFlags: input.FeatureFlags, Plugins: resolvePlugins(input.Plugins)}
}

// PluginConfig mirrors a resolved plugin entry { resolve, options }.
type PluginConfig struct {
	Resolve string         `json:"resolve" yaml:"resolve"`
	Options map[string]any `json:"options,omitempty" yaml:"options,omitempty"`
}

// resolvePlugins mirrors resolvePlugins: the default @medusajs/draft-order
// plugin, then the configured plugins (by resolve, the last one wins).
func resolvePlugins(configPlugins []PluginConfig) []PluginConfig {
	out := []PluginConfig{{Resolve: "@medusajs/draft-order", Options: map[string]any{}}}
	for _, p := range configPlugins {
		if p.Options == nil {
			p.Options = map[string]any{}
		}
		replaced := false
		for i := range out {
			if out[i].Resolve == p.Resolve {
				out[i] = p
				replaced = true
			}
		}
		if !replaced {
			out = append(out, p)
		}
	}
	return out
}

const (
	defaultStoreCors = "http://localhost:8000"
	defaultAdminCors = "http://localhost:7000,http://localhost:7001,http://localhost:5173"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mod(key string, providers ...ModuleProviderConfig) ModuleConfig {
	return ModuleConfig{Key: key, Resolve: modulessdk.MODULE_PACKAGE_NAMES[key], Providers: providers}
}

// featureFlagEnabled resolves a feature flag like registerFeatureFlag
// (environment first, then projectConfig.featureFlags).
//
// Upstream evaluates FeatureFlag.isFeatureEnabled while the configuration
// file is loaded, before the core flags are registered, so a stock
// configuration keeps the translation module disabled whatever the flag;
// the documented setup adds the module and sets the flag. Here the flag
// alone enables the module (the reference application adds the module when
// MEDUSA_FF_TRANSLATION is set).
func featureFlagEnabled(featureFlags map[string]any, key, envKey string) bool {
	router := ff.NewFlagRouter()
	ff.RegisterFeatureFlag(ff.FlagSettings{Key: key, EnvKey: envKey}, featureFlags, router)
	return router.IsFeatureEnabled(key)
}

// defaultModules mirrors the self-hosted default modules of define-config.ts.
func defaultModules(featureFlags map[string]any) []ModuleConfig {
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
			Disable: !featureFlagEnabled(featureFlags, "translation", "MEDUSA_FF_TRANSLATION"),
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

// DefaultStoreRestrictedFields mirrors DEFAULT_STORE_RESTRICTED_FIELDS.
var DefaultStoreRestrictedFields = []string{"order", "orders"}

// IsProduction mirrors isProduction: NODE_ENV=production.
func IsProduction() bool { return os.Getenv("NODE_ENV") == "production" }

// IsStaging mirrors the NODE_ENV=staging check of the express loader.
func IsStaging() bool { return os.Getenv("NODE_ENV") == "staging" }
