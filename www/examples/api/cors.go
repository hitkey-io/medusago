package api

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// region: config
// medusaConfig is the project configuration (medusa-config.go). The CORS
// settings accept a comma-separated list of origins, or a regular expression
// between slashes.
func medusaConfig() config.InputConfig {
	return config.InputConfig{
		ProjectConfig: config.ProjectConfig{
			HTTP: config.HttpConfig{
				StoreCors: "http://localhost:8000",
				AdminCors: "http://localhost:7001",
			},
		},
	}
}

// endregion: config

// region: disable
func storeCustomGET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return res.JSON(map[string]any{"message": "[GET] Hello world!"})
}

// StoreCustomNoCorsRoute disables the CORS middleware for /store/custom-hello.
var StoreCustomNoCorsRoute = medusahttp.RouteFile{
	Path: "/store/custom-hello",
	Route: medusahttp.Route{
		GET:  storeCustomGET,
		Cors: ptr(false),
	},
}

// endregion: disable

// region: custom-routes
// corsForCustomRoutes applies the storeCors configuration to the routes it
// is registered for.
func corsForCustomRoutes(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
	cfg, _ := req.Scope.Resolve(common.ContainerRegistrationKeys.CONFIG_MODULE).(config.ConfigModule)

	origins := medusahttp.ParseCorsOrigins(cfg.ProjectConfig.HTTP.StoreCors)
	return medusahttp.Cors(origins)(req, res, next)
}

// CorsMiddleware applies CORS to every route starting with /custom.
var CorsMiddleware = medusahttp.MiddlewareRoute{
	Matcher:     "/custom*",
	Middlewares: []medusahttp.MiddlewareFunction{corsForCustomRoutes},
}

// endregion: custom-routes
