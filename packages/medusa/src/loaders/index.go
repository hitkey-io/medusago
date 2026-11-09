package loaders

import (
	"context"
	"fmt"
	"log/slog"
	nethttp "net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/framework/src/container"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/api"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
)

// MedusaApp is the loaded application.
type MedusaApp struct {
	Container *container.MedusaContainer
	Handler   nethttp.Handler
	Pool      *pgxpool.Pool
}

// Shutdown releases the resources.
func (a *MedusaApp) Shutdown() {
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
		exp, ok := modules.Registry[mc.Resolve]
		if !ok {
			continue
		}
		var providers []modulessdk.ProviderInstance
		for _, p := range mc.Providers {
			pi := modulessdk.ProviderInstance{ID: p.ID, Options: p.Options}
			if pexp, ok := modules.Registry[p.Resolve]; ok {
				pi.Identifier = pexp.ProviderIdentifier
				if pexp.NewProvider != nil {
					inst, err := pexp.NewProvider(modulessdk.ProviderOptions{ID: p.ID, Options: p.Options, Logger: logger, Container: c})
					if err != nil {
						return fmt.Errorf("%s provider %s: %w", mc.Key, p.ID, err)
					}
					pi.Instance = inst
				}
			}
			providers = append(providers, pi)
		}
		var svc any
		switch {
		case exp.Service != nil:
			s, err := exp.Service(modulessdk.ServiceOptions{DB: pool, Providers: providers, Logger: logger, Container: c})
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
		switch v := co.SameSite.(type) {
		case string:
			o.Cookie.SameSite = v
		case bool:
			if v {
				o.Cookie.SameSite = "strict"
			} else {
				o.Cookie.SameSite = ""
			}
		}
	}
	return o
}

// Initialize mirrors the loaders of `medusa start`: container, modules,
// Query, the express loader and the API loader.
func Initialize(ctx context.Context, cfg config.ConfigModule) (*MedusaApp, error) {
	logger := slog.Default()
	pool, err := pgxpool.New(ctx, cfg.ProjectConfig.DatabaseURL)
	if err != nil {
		return nil, err
	}
	c := container.New()
	c.Register(common.ContainerRegistrationKeys.CONFIG_MODULE, cfg)
	c.Register(common.ContainerRegistrationKeys.LOGGER, logger)
	c.Register(common.ContainerRegistrationKeys.PG_CONNECTION, pool)

	if err := LoadModules(ctx, c, pool, cfg, logger); err != nil {
		pool.Close()
		return nil, err
	}
	q, err := NewDefaultQuery(pool)
	if err != nil {
		pool.Close()
		return nil, err
	}
	c.Register(common.ContainerRegistrationKeys.QUERY, q)

	app := medusahttp.NewApp()
	app.NewRequest = func(r *nethttp.Request) *medusahttp.MedusaRequest {
		req := medusahttp.NewMedusaRequest(r)
		req.Scope = c
		req.Query = medusahttp.ParseQS(r.URL.RawQuery)
		return req
	}
	// express loader
	app.Use("/", medusahttp.SessionMiddleware(sessionOptions(cfg.ProjectConfig)))
	// start.ts registers the health check after the loaders; it is matched
	// by path only, so its position does not change the routing.
	medusahttp.LoadAPI(app, medusahttp.ApiLoaderOptions{
		Routes:               api.Routes(),
		Middlewares:          api.Middlewares(),
		AdminCors:            cfg.ProjectConfig.HTTP.AdminCors,
		StoreCors:            cfg.ProjectConfig.HTTP.StoreCors,
		AuthCors:             cfg.ProjectConfig.HTTP.AuthCors,
		Authenticate:         middlewares.Authenticate,
		BodyParser:           middlewares.BodyParser,
		BaseRestrictedFields: cfg.ProjectConfig.HTTP.RestrictedFields.Store,
		StoreRelationsLimit:  cfg.ProjectConfig.HTTP.StoreRelationsLimit,
		CoreErrorHandler:     middlewares.ErrorHandler(),
	})
	app.Handle("GET", "/health", func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, _ medusahttp.NextFunction) error {
		res.Status(200).Send("OK")
		return nil
	})
	return &MedusaApp{Container: c, Handler: app, Pool: pool}, nil
}
