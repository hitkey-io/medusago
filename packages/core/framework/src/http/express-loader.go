package http

import "strings"

// SecurityHeadersOptions mirrors HttpSecurityHeadersOptions
// (projectConfig.http.securityHeaders).
type SecurityHeadersOptions struct {
	// Enabled defaults to true.
	Enabled *bool
	// FrameAncestors defaults to ["'self'"]; when set (non-nil), the
	// X-Frame-Options header is not sent.
	FrameAncestors []string
}

// ApplySecurityHeaders mirrors applySecurityHeaders (express-loader.ts):
// X-Powered-By is always disabled and, unless disabled, every response gets
// X-Content-Type-Options, X-Frame-Options (without custom frame ancestors)
// and Content-Security-Policy: frame-ancestors. It must be registered before
// any other middleware.
func ApplySecurityHeaders(app *App, options *SecurityHeadersOptions) {
	app.XPoweredBy = false
	if options == nil {
		options = &SecurityHeadersOptions{}
	}
	if options.Enabled != nil && !*options.Enabled {
		return
	}
	frameAncestors := options.FrameAncestors
	if frameAncestors == nil {
		frameAncestors = []string{"'self'"}
	}
	contentSecurityPolicy := "frame-ancestors " + strings.Join(frameAncestors, " ")
	customAncestors := options.FrameAncestors != nil
	app.Use("/", func(_ *MedusaRequest, res *MedusaResponse, next NextFunction) error {
		res.SetHeader("X-Content-Type-Options", "nosniff")
		if !customAncestors {
			res.SetHeader("X-Frame-Options", "SAMEORIGIN")
		}
		res.SetHeader("Content-Security-Policy", contentSecurityPolicy)
		next(nil)
		return nil
	})
}
