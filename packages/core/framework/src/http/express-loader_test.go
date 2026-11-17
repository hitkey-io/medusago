package http

import (
	"net/http/httptest"
	"testing"
)

func securityApp(opts *SecurityHeadersOptions) *App {
	app := NewApp()
	ApplySecurityHeaders(app, opts)
	app.Handle("GET", "/health", func(_ *MedusaRequest, res *MedusaResponse, _ NextFunction) error {
		res.SendStatus(200)
		return nil
	})
	return app
}

func TestApplySecurityHeaders(t *testing.T) {
	disabled := false
	cases := []struct {
		name                string
		opts                *SecurityHeadersOptions
		nosniff, frame, csp string
	}{
		{"defaults", nil, "nosniff", "SAMEORIGIN", "frame-ancestors 'self'"},
		{"frame ancestors", &SecurityHeadersOptions{FrameAncestors: []string{"'self'", "https://dashboard.example.com"}}, "nosniff", "", "frame-ancestors 'self' https://dashboard.example.com"},
		{"disabled", &SecurityHeadersOptions{Enabled: &disabled}, "", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			securityApp(c.opts).ServeHTTP(w, httptest.NewRequest("GET", "/health", nil))
			h := w.Header()
			if h.Get("X-Powered-By") != "" {
				t.Errorf("X-Powered-By = %q, want none", h.Get("X-Powered-By"))
			}
			if got := h.Get("X-Content-Type-Options"); got != c.nosniff {
				t.Errorf("X-Content-Type-Options = %q, want %q", got, c.nosniff)
			}
			if got := h.Get("X-Frame-Options"); got != c.frame {
				t.Errorf("X-Frame-Options = %q, want %q", got, c.frame)
			}
			if got := h.Get("Content-Security-Policy"); got != c.csp {
				t.Errorf("Content-Security-Policy = %q, want %q", got, c.csp)
			}
		})
	}
}

func TestSecurityHeadersOnNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	securityApp(nil).ServeHTTP(w, httptest.NewRequest("GET", "/missing", nil))
	h := w.Header()
	for k, want := range map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "SAMEORIGIN",
		"Content-Security-Policy": "default-src 'none'",
	} {
		if got := h.Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
}

func TestCorsOptions(t *testing.T) {
	maxAge := 7200.0
	app := NewApp()
	app.Use("/store", Cors(ParseCorsOrigins("http://localhost:8000"), CorsOptions{
		MaxAge: &maxAge, ExposedHeaders: []string{"X-A", "X-B"}, AllowedHeaders: "Content-Type", Methods: []any{"GET", "POST"},
	}))
	req := httptest.NewRequest("OPTIONS", "/store/products", nil)
	req.Header.Set("Origin", "http://localhost:8000")
	req.Header.Set("Access-Control-Request-Headers", "x-custom")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	h := w.Header()
	for k, want := range map[string]string{
		"Access-Control-Allow-Origin":   "http://localhost:8000",
		"Access-Control-Allow-Methods":  "GET,POST",
		"Access-Control-Allow-Headers":  "Content-Type",
		"Access-Control-Max-Age":        "7200",
		"Access-Control-Expose-Headers": "X-A,X-B",
		"Vary":                          "Origin",
	} {
		if got := h.Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
	if w.Code != 204 {
		t.Errorf("status = %d, want 204", w.Code)
	}
}
