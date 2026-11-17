package loaders

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
)

// securityHeadersOptions maps projectConfig.http.securityHeaders.
func securityHeadersOptions(o *config.HttpSecurityHeadersOptions) *medusahttp.SecurityHeadersOptions {
	if o == nil {
		return nil
	}
	return &medusahttp.SecurityHeadersOptions{Enabled: o.Enabled, FrameAncestors: o.FrameAncestors}
}

// corsOptions maps projectConfig.http.corsOptions.
func corsOptions(o *config.HttpCorsOptions) medusahttp.CorsOptions {
	if o == nil {
		return medusahttp.CorsOptions{}
	}
	return medusahttp.CorsOptions{MaxAge: o.MaxAge, ExposedHeaders: o.ExposedHeaders, AllowedHeaders: o.AllowedHeaders, Methods: o.Methods}
}
