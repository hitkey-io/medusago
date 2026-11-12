package loaders

import (
	"context"
	"fmt"
	defaultsworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/defaults/workflows"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/medusa/src/subscribers"
	"log/slog"
	nethttp "net/http"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/framework/src/container"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	ff "github.com/hitkey-io/medusago/packages/core/utils/src/feature-flags"
	"github.com/hitkey-io/medusago/packages/core/utils/src/ioredis"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/api"
	featureflags "github.com/hitkey-io/medusago/packages/medusa/src/feature-flags"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
)

// MedusaApp is the loaded application.
type MedusaApp struct {
	Container *container.MedusaContainer
	Handler   nethttp.Handler
	// App is the router behind Handler (routes and middlewares can be
	// added to it by Options.OnLoad hooks).
	App  *medusahttp.App
	Pool *pgxpool.Pool
	// shutdown mirrors the shutdown returned by the express loader.
	shutdown []func()
}

// Shutdown releases the resources.
func (a *MedusaApp) Shutdown() {
	for _, fn := range a.shutdown {
		fn()
	}
	if a.Pool != nil {
		a.Pool.Close()
	}
}

// LoadModules mirrors loadModules: each enabled module service is
// registered in the container under its key.
func LoadModules(ctx context.Context, c *container.MedusaContainer, pool *pgxpool.Pool, cfg config.ConfigModule, logger *slog.Logger) error {
	for _, mc := range cfg.Modules {
		if mc.Disable {
			continue
		}
		exp, ok := modules.Lookup(mc)
		if !ok {
			continue
		}
		providers, err := moduleProviders(mc, logger, c)
		if err != nil {
			return err
		}
		var svc any
		switch {
		case exp.Service != nil:
			s, err := exp.Service(modulessdk.ServiceOptions{DB: pool, Providers: providers, Options: mc.Options, Logger: logger, Container: c})
			if err != nil {
				return fmt.Errorf("%s: %w", mc.Key, err)
			}
			svc = s
		case len(exp.Models) > 0:
			svc = utilsmodulessdk.NewMedusaService(pool, exp.Models, exp.JoinerConfig)
		default:
			continue
		}
		c.Register(mc.Key, svc)
	}
	// module services emit their events to the event bus module
	// (eventBusModuleService_, resolved from the module container)
	if bus, ok := c.Resolve(utilsmodulessdk.EVENT_BUS).(eventbus.Emitter); ok {
		for _, mc := range cfg.Modules {
			if s, ok := c.Resolve(mc.Key).(interface{ SetEventBus(eventbus.Emitter) }); ok {
				s.SetEventBus(bus)
			}
		}
	}
	return nil
}

// moduleProviders builds the configured providers of a module: one instance
// per service of each provider package (moduleProviderLoader).
func moduleProviders(mc config.ModuleConfig, logger *slog.Logger, c *container.MedusaContainer) ([]modulessdk.ProviderInstance, error) {
	var providers []modulessdk.ProviderInstance
	for _, p := range mc.Providers {
		pexp, ok := modules.Registry[p.Resolve]
		if !ok {
			providers = append(providers, modulessdk.ProviderInstance{ID: p.ID, Options: p.Options, IsDefault: p.IsDefault})
			continue
		}
		for _, ps := range pexp.Services() {
			pi := modulessdk.ProviderInstance{Identifier: ps.Identifier, ID: p.ID, Options: p.Options, IsDefault: p.IsDefault}
			if ps.ValidateOptions != nil {
				if err := ps.ValidateOptions(p.Options); err != nil {
					return nil, err
				}
			}
			if ps.NewProvider != nil {
				inst, err := ps.NewProvider(modulessdk.ProviderOptions{ID: p.ID, Options: p.Options, Logger: logger, Container: c})
				if err != nil {
					return nil, fmt.Errorf("%s provider %s: %w", mc.Key, p.ID, err)
				}
				pi.Instance = inst
			}
			providers = append(providers, pi)
		}
	}
	return providers, nil
}

// runStartHooks runs, for the enabled modules, the loaders that also run at
// start (LoadersOnStart), then the onApplicationStart hooks.
func runStartHooks(ctx context.Context, c *container.MedusaContainer, pool *pgxpool.Pool, cfg config.ConfigModule, logger *slog.Logger) error {
	type started struct {
		key string
		exp modulessdk.ModuleExports
	}
	var enabled []started
	for _, mc := range cfg.Modules {
		if mc.Disable {
			continue
		}
		exp, ok := modules.Lookup(mc)
		if !ok {
			continue
		}
		enabled = append(enabled, started{key: mc.Key, exp: *exp})
		if !exp.LoadersOnStart {
			continue
		}
		var providers []modulessdk.ProviderInstance
		for _, p := range mc.Providers {
			if pexp, ok := modules.Registry[p.Resolve]; ok {
				for _, ps := range pexp.Services() {
					providers = append(providers, modulessdk.ProviderInstance{Identifier: ps.Identifier, ID: p.ID, Options: p.Options})
				}
			}
		}
		for _, load := range exp.Loaders {
			if err := load(ctx, modulessdk.LoaderOptions{DB: pool, Providers: providers, Options: mc.Options, Logger: logger}); err != nil {
				return fmt.Errorf("%s loader: %w", mc.Key, err)
			}
		}
	}
	for _, m := range enabled {
		if m.exp.OnApplicationStart == nil {
			continue
		}
		if err := m.exp.OnApplicationStart(ctx, c.Resolve(m.key)); err != nil {
			return fmt.Errorf("%s onApplicationStart: %w", m.key, err)
		}
	}
	return nil
}

// sessionOptions mirrors the session options of the express loader.
func sessionOptions(cfg config.ProjectConfig) medusahttp.SessionOptions {
	o := medusahttp.SessionOptions{
		Name:              "connect.sid",
		Resave:            true,
		Rolling:           false,
		SaveUninitialized: false,
		Secret:            cfg.HTTP.CookieSecret,
		Cookie: medusahttp.SessionCookieOptions{
			Path:     "/",
			HttpOnly: true,
			MaxAge:   10 * 60 * 60 * 1000,
		},
	}
	if config.IsProduction() || config.IsStaging() {
		o.Cookie.SameSite = "lax"
		o.Cookie.Secure = true
	}
	if so := cfg.SessionOptions; so != nil {
		if so.Name != "" {
			o.Name = so.Name
		}
		if so.Resave != nil {
			o.Resave = *so.Resave
		}
		if so.Rolling != nil {
			o.Rolling = *so.Rolling
		}
		if so.SaveUninitialized != nil {
			o.SaveUninitialized = *so.SaveUninitialized
		}
		if so.Secret != "" {
			o.Secret = so.Secret
		}
		if so.Ttl != nil {
			o.Cookie.MaxAge = *so.Ttl
		}
	}
	if co := cfg.CookieOptions; co != nil {
		if co.Secure != nil {
			o.Cookie.Secure = *co.Secure
		}
		if co.HttpOnly != nil {
			o.Cookie.HttpOnly = *co.HttpOnly
		}
		if co.Domain != "" {
			o.Cookie.Domain = co.Domain
		}
		if co.Path != "" {
			o.Cookie.Path = co.Path
		}
		if co.MaxAge != nil {
			o.Cookie.MaxAge = *co.MaxAge
		}
		if co.Partitioned != nil {
			o.Cookie.Partitioned = *co.Partitioned
		}
		if co.Priority != "" {
			o.Cookie.Priority = co.Priority
		}
		switch v := co.SameSite.(type) {
		case string:
			o.Cookie.SameSite = v
		case bool:
			if v {
				o.Cookie.SameSite = "true"
			} else {
				o.Cookie.SameSite = ""
			}
		}
	}
	return o
}

// sessionStore mirrors the store selection of the express loader: with
// sessionOptions.dynamodbOptions, connect-dynamodb; with
// projectConfig.redisUrl, connect-redis over an ioredis client created with
// redisOptions and the key prefix `${redisPrefix ?? ""}sess:`; otherwise
// express-session's MemoryStore. The returned func disconnects the client.
func sessionStore(cfg config.ProjectConfig) (medusahttp.SessionStore, func(), error) {
	if cfg.SessionOptions != nil && cfg.SessionOptions.DynamodbOptions != nil {
		store, err := medusahttp.NewDynamoDBStore(cfg.SessionOptions.DynamodbOptions, nil)
		if err != nil {
			return nil, nil, err
		}
		return store, store.Close, nil
	}
	if cfg.RedisURL == "" {
		return medusahttp.NewMemoryStore(), func() {}, nil
	}
	client := ioredis.New(cfg.RedisURL, cfg.RedisOptions)
	prefix := cfg.RedisPrefix + "sess:"
	store := medusahttp.NewRedisStore(medusahttp.RedisStoreOptions{Client: client, Prefix: &prefix})
	return store, func() { _ = client.Close() }, nil
}

// applicationShutdowner is a module service with an onApplicationShutdown
// hook (Redis connections of the locking and cache modules).
type applicationShutdowner interface {
	OnApplicationShutdown(ctx context.Context) error
}

// moduleShutdowns returns the onApplicationShutdown hooks of the loaded
// modules.
func moduleShutdowns(c *container.MedusaContainer, cfg config.ConfigModule) []func() {
	var out []func()
	for _, mc := range cfg.Modules {
		if mc.Disable {
			continue
		}
		if s, ok := c.Resolve(mc.Key).(applicationShutdowner); ok {
			out = append(out, func() { _ = s.OnApplicationShutdown(context.Background()) })
		}
	}
	return out
}

// applicationStarter is a module service with an onApplicationStart hook.
type applicationStarter interface {
	OnApplicationStart(ctx context.Context) error
}

// Options are the options of InitializeWithOptions.
type Options struct {
	// Directory is the project root (static files, admin build); defaults
	// to the working directory.
	Directory string
	// SkipAdmin does not register the admin dashboard (like
	// skipLoadingEntryPoints for `medusa exec`).
	SkipAdmin bool
	// OnLoad runs after the application is loaded, before it serves
	// requests: the hook of a project to register its own routes,
	// subscribers or container entries.
	OnLoad []func(ctx context.Context, app *MedusaApp) error
	// Routes and Middlewares are the API routes and middlewares of the
	// project (src/api), loaded after the core ones like upstream.
	Routes      []medusahttp.RouteFile
	Middlewares []medusahttp.MiddlewareRoute
}

// Initialize mirrors the loaders of `medusa start`: container, modules,
// Query, the express loader, the admin loader and the API loader.
func Initialize(ctx context.Context, cfg config.ConfigModule) (*MedusaApp, error) {
	return InitializeWithOptions(ctx, cfg, Options{})
}

// InitializeWithOptions is Initialize with options.
func InitializeWithOptions(ctx context.Context, cfg config.ConfigModule, opts Options) (*MedusaApp, error) {
	logger := slog.Default()
	if opts.Directory == "" {
		if cwd, err := os.Getwd(); err == nil {
			opts.Directory = cwd
		}
	}
	pool, err := pgxpool.New(ctx, cfg.ProjectConfig.DatabaseURL)
	if err != nil {
		return nil, err
	}
	c := container.New()
	c.Register(common.ContainerRegistrationKeys.CONFIG_MODULE, cfg)
	c.Register(common.ContainerRegistrationKeys.LOGGER, logger)
	c.Register(common.ContainerRegistrationKeys.PG_CONNECTION, pool)

	flagRouter := ff.NewFlagRouter()
	for _, f := range featureflags.All {
		ff.RegisterFeatureFlag(f, cfg.FeatureFlags, flagRouter)
	}
	c.Register(common.ContainerRegistrationKeys.FEATURE_FLAG_ROUTER, flagRouter)

	if err := LoadModules(ctx, c, pool, cfg, logger); err != nil {
		pool.Close()
		return nil, err
	}
	moduleServices := map[string]any{}
	for _, mc := range cfg.Modules {
		if v := c.Resolve(mc.Key); v != nil {
			moduleServices[mc.Key] = v
		}
	}
	q, link, _, err := NewQueryAndLink(pool, cfg, moduleServices)
	if err != nil {
		pool.Close()
		return nil, err
	}
	c.Register(common.ContainerRegistrationKeys.QUERY, q)
	c.Register(common.ContainerRegistrationKeys.LINK, link)
	c.Register(common.ContainerRegistrationKeys.REMOTE_LINK, link)

	if err := runStartHooks(ctx, c, pool, cfg, logger); err != nil {
		pool.Close()
		return nil, err
	}

	// subscribers loader (packages/medusa/src/subscribers)
	if bus, ok := c.Resolve(utilsmodulessdk.EVENT_BUS).(eventbus.Subscribable); ok {
		for _, event := range subscribers.ConfigurableNotificationsEvents() {
			bus.Subscribe(event, func(ctx context.Context, ev eventbus.Event) error {
				return subscribers.ConfigurableNotificationsHandler(ctx, c, ev.Name, ev.Data)
			})
		}
		bus.Subscribe("payment.webhook_received", func(ctx context.Context, ev eventbus.Event) error {
			return subscribers.PaymentWebhookHandler(ctx, c, ev.Data)
		})
		for _, event := range subscribers.SearchIngestionEvents() {
			bus.Subscribe(event, func(ctx context.Context, ev eventbus.Event) error {
				return subscribers.SearchIngestionHandler(ctx, c, ev.Name, ev.Data, ev.Metadata)
			})
		}
	}

	// createQuery's indexModule and searchModule
	if index, ok := c.Resolve(utilsmodulessdk.INDEX).(query.IndexService); ok {
		q.SetIndexModule(index)
	}
	if search, ok := c.Resolve(utilsmodulessdk.SEARCH).(query.SearchService); ok {
		q.SetSearchModule(search)
	}
	// @Cached resolves the caching module (feature flag "caching")
	if cm, ok := c.Resolve(utilsmodulessdk.CACHING).(caching.Module); ok {
		q.SetCaching(cm, flagRouter.IsFeatureEnabled(featureflags.Caching.Key))
	}
	// loaders/index.ts: createDefaultsWorkflow runs at every start, before
	// the modules' onApplicationStart hooks.
	if _, err := defaultsworkflows.CreateDefaultsWorkflow.Run(ctx, c, nil); err != nil {
		pool.Close()
		return nil, err
	}
	// MedusaApp onApplicationStart: the modules' hooks (the index module
	// creates its partitions and synchronises the changed entities).
	for _, mc := range cfg.Modules {
		if mc.Disable {
			continue
		}
		if starter, ok := c.Resolve(mc.Key).(applicationStarter); ok {
			if err := starter.OnApplicationStart(ctx); err != nil {
				pool.Close()
				return nil, err
			}
		}
	}

	app := medusahttp.NewApp()
	app.NewRequest = func(r *nethttp.Request) *medusahttp.MedusaRequest {
		req := medusahttp.NewMedusaRequest(r)
		req.Scope = c
		req.Query = medusahttp.ParseQS(r.URL.RawQuery)
		return req
	}
	// express loader
	sessOpts := sessionOptions(cfg.ProjectConfig)
	store, closeStore, err := sessionStore(cfg.ProjectConfig)
	if err != nil {
		return nil, err
	}
	sessOpts.Store = store
	app.Use("/", medusahttp.SessionMiddleware(sessOpts))
	if opts.Directory != "" {
		app.Use("/static", medusahttp.Static("/static", filepath.Join(opts.Directory, "static")))
	}
	// start.ts registers the health check after the loaders; it is matched
	// by path only, so its position does not change the routing.
	medusahttp.LoadAPI(app, medusahttp.ApiLoaderOptions{
		Routes:               append(api.Routes(), opts.Routes...),
		Middlewares:          append(api.Middlewares(), opts.Middlewares...),
		AdminCors:            cfg.ProjectConfig.HTTP.AdminCors,
		StoreCors:            cfg.ProjectConfig.HTTP.StoreCors,
		AuthCors:             cfg.ProjectConfig.HTTP.AuthCors,
		Authenticate:         middlewares.Authenticate,
		BodyParser:           middlewares.BodyParser,
		BaseRestrictedFields: cfg.ProjectConfig.HTTP.RestrictedFields.Store,
		StoreRelationsLimit:  cfg.ProjectConfig.HTTP.StoreRelationsLimit,
		CoreErrorHandler:     middlewares.ErrorHandler(),
		StorePublishableKey:  middlewares.EnsurePublishableApiKey,
		StoreLocale:          middlewares.ApplyLocale,
		IsFeatureEnabled:     flagRouter.IsFeatureEnabled,
	})
	app.Handle("GET", "/health", func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, _ medusahttp.NextFunction) error {
		res.Status(200).Send("OK")
		return nil
	})
	medusaApp := &MedusaApp{Container: c, Handler: app, App: app, Pool: pool, shutdown: append([]func(){closeStore}, moduleShutdowns(c, cfg)...)}
	// admin loader (upstream registers it once the bundler is imported, after
	// the synchronous registrations).
	if !opts.SkipAdmin {
		if err := AdminLoader(app, cfg, opts.Directory, logger); err != nil {
			medusaApp.Shutdown()
			return nil, err
		}
	}
	for _, hook := range opts.OnLoad {
		if err := hook(ctx, medusaApp); err != nil {
			medusaApp.Shutdown()
			return nil, err
		}
	}
	return medusaApp, nil
}
