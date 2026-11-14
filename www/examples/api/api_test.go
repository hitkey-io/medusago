package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/auth"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	apiutils "github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

const jwtSecret = "test-secret"

type container map[string]any

func (c container) Resolve(key string) any { return c[key] }

// newServer loads the routes and middlewares of the example like the
// application does, without the modules.
func newServer(t *testing.T, routes []medusahttp.RouteFile, mws []medusahttp.MiddlewareRoute) *httptest.Server {
	t.Helper()
	cfg := config.DefineConfig(config.InputConfig{ProjectConfig: config.ProjectConfig{
		HTTP: config.HttpConfig{StoreCors: "http://localhost:8000", JwtSecret: jwtSecret},
	}})
	scope := container{common.ContainerRegistrationKeys.CONFIG_MODULE: cfg}

	app := medusahttp.NewApp()
	app.NewRequest = func(r *http.Request) *medusahttp.MedusaRequest {
		req := medusahttp.NewMedusaRequest(r)
		req.Scope = scope
		req.Query = medusahttp.ParseQS(r.URL.RawQuery)
		return req
	}
	medusahttp.LoadAPI(app, medusahttp.ApiLoaderOptions{
		Routes:               routes,
		Middlewares:          mws,
		AdminCors:            cfg.ProjectConfig.HTTP.AdminCors,
		StoreCors:            cfg.ProjectConfig.HTTP.StoreCors,
		AuthCors:             cfg.ProjectConfig.HTTP.AuthCors,
		Authenticate:         middlewares.Authenticate,
		BodyParser:           middlewares.BodyParser,
		BaseRestrictedFields: cfg.ProjectConfig.HTTP.RestrictedFields.Store,
		StoreRelationsLimit:  cfg.ProjectConfig.HTTP.StoreRelationsLimit,
		CoreErrorHandler:     middlewares.ErrorHandler(),
	})
	srv := httptest.NewServer(app)
	t.Cleanup(srv.Close)
	return srv
}

func do(t *testing.T, method, url, body string, headers map[string]string) (int, string, http.Header) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, strings.TrimSpace(string(raw)), resp.Header
}

func TestExampleRoutes(t *testing.T) {
	srv := newServer(t, Routes, Middlewares)

	token, err := auth.GenerateJwtToken(func() *dal.Record {
		r := dal.NewRecord()
		r.Set("actor_id", "user_01")
		r.Set("actor_type", "user")
		r.Set("auth_identity_id", "authid_01")
		return r
	}(), jwtSecret, "1h", auth.JwtOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(`{"event":"created"}`))
	signature := hex.EncodeToString(mac.Sum(nil))

	cases := []struct {
		name    string
		method  string
		path    string
		body    string
		headers map[string]string
		status  int
		want    string
	}{
		{"get", "GET", "/hello-world", "", nil, 200, `{"message":"[GET] Hello world!"}`},
		{"post", "POST", "/hello-world", "", nil, 200, `{"message":"[POST] Hello world!"}`},
		{"path param", "GET", "/hello-world/123", "", nil, 200, `{"message":"[GET] Hello 123!"}`},
		{"path params", "GET", "/hello-world/123/name/john", "", nil, 200, `{"message":"[GET] Hello 123 - john!"}`},
		{"query param", "GET", "/greeting?name=John", "", nil, 200, `{"message":"Hello John"}`},
		{"body param", "POST", "/greeting", `{"name":"John"}`, nil, 200, `{"message":"[POST] Hello John!"}`},
		{"typed body", "POST", "/custom/posts", `{"title":"Hello","author_email":"a@example.com"}`, nil, 201, `{"status":"draft","title":"Hello"}`},
		{"invalid email", "POST", "/custom/posts", `{"title":"Hello","author_email":"nope"}`, nil, 400, `{"type":"invalid_data","message":"Invalid request: Invalid email address"}`},
		{"unknown property", "POST", "/custom/sum", `{"a":1,"b":2,"c":3}`, nil, 400, `{"type":"invalid_data","message":"Invalid request: Unrecognized fields: 'c'"}`},
		{"status", "POST", "/custom", "", nil, 201, `{"message":"Created!"}`},
		{"body validation", "POST", "/custom/sum", `{"a":1,"b":2}`, nil, 200, `{"sum":3}`},
		{"body validation error", "POST", "/custom/sum", `{"b":2}`, nil, 400, `{"type":"invalid_data","message":"Invalid request: Field 'a' is required"}`},
		{"query validation", "GET", "/custom/sum?a=1&b=2", "", nil, 200, `{"sum":3}`},
		{"query validation error", "GET", "/custom/sum?b=2", "", nil, 400, `{"type":"invalid_data","message":"Invalid request: Field 'a' is required"}`},
		{"error", "GET", "/custom/search", "", nil, 400, `{"type":"invalid_data","message":"The ` + "`q`" + ` query parameter is required."}`},
		{"error code", "GET", "/custom/quota", "", nil, 400, `{"code":"quota_reached","type":"not_allowed","message":"The quota was reached."}`},
		{"unauthenticated", "GET", "/custom/admin/whoami", "", nil, 401, `{"message":"Unauthorized"}`},
		{"authenticated", "GET", "/custom/admin/whoami", "", map[string]string{"Authorization": "Bearer " + token}, 200, `{"actor_id":"user_01","actor_type":"user"}`},
		{"customer route rejects a user token", "GET", "/custom/customer/profile", "", map[string]string{"Authorization": "Bearer " + token}, 401, `{"message":"Unauthorized"}`},
		{"opt out of authentication", "GET", "/admin/custom", "", nil, 200, `{"message":"Hello"}`},
		{"admin routes are protected", "GET", "/admin/users", "", nil, 401, `{"message":"Unauthorized"}`},
		{"raw body", "POST", "/webhooks/custom", `{"event":"created"}`, map[string]string{"x-signature": signature}, 200, `{"received":19}`},
		{"raw body, wrong signature", "POST", "/webhooks/custom", `{"event":"created"}`, map[string]string{"x-signature": "nope"}, 401, `{"type":"unauthorized","message":"Invalid signature."}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, body, _ := do(t, c.method, srv.URL+c.path, c.body, c.headers)
			if status != c.status || body != c.want {
				t.Fatalf("%s %s: got %d %s, want %d %s", c.method, c.path, status, body, c.status, c.want)
			}
		})
	}
}

func TestCors(t *testing.T) {
	srv := newServer(t, Routes, Middlewares)

	status, _, h := do(t, "OPTIONS", srv.URL+"/custom/search", "", map[string]string{
		"Origin":                        "http://localhost:8000",
		"Access-Control-Request-Method": "GET",
	})
	if status != 204 || h.Get("Access-Control-Allow-Origin") != "http://localhost:8000" {
		t.Fatalf("preflight: %d %v", status, h)
	}

	_, _, h = do(t, "GET", srv.URL+"/custom/search?q=a", "", map[string]string{"Origin": "http://evil.example"})
	if h.Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("an origin that isn't configured must not be allowed: %v", h)
	}

	// storeCors is applied to /store routes, unless the route disables it
	srv2 := newServer(t, []medusahttp.RouteFile{StoreCustomNoCorsRoute}, nil)
	_, _, h = do(t, "GET", srv2.URL+"/store/custom-hello", "", map[string]string{"Origin": "http://localhost:8000"})
	if h.Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("CORS must be disabled for the route: %v", h)
	}
}

// echoFields serves GET /store/custom with a query config, and returns what
// the config resolved instead of querying the database.
func echoFields(qc httputils.QueryConfig) (medusahttp.RouteFile, medusahttp.MiddlewareRoute) {
	route := medusahttp.RouteFile{Path: "/store/custom", Route: medusahttp.Route{GET: func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
		return res.JSON(map[string]any{"fields": req.QueryConfig.Fields, "limit": req.QueryConfig.Pagination})
	}}}
	mw := medusahttp.MiddlewareRoute{Matcher: "/store/custom", Methods: []string{"GET"}, Middlewares: []medusahttp.MiddlewareFunction{
		httputils.ValidateAndTransformQuery(apiutils.CreateFindParams(apiutils.FindParamsOptions{}), qc),
	}}
	return route, mw
}

func TestAllowedFields(t *testing.T) {
	route, mw := echoFields(ProductListConfig)
	srv := newServer(t, []medusahttp.RouteFile{route}, []medusahttp.MiddlewareRoute{mw})

	cases := []struct {
		query  string
		status int
		want   string
	}{
		{"", 200, `{"fields":["id","title"],"limit":{"skip":0,"take":20}}`},
		{"?fields=id,title,handle", 200, `{"fields":["id","title","handle"],"limit":{"skip":0,"take":20}}`},
		// fields that aren't allowed are removed silently
		{"?fields=id,title,metadata", 200, `{"fields":["id","title"],"limit":{"skip":0,"take":20}}`},
		// a relation is allowed on its own, not the fields nested under it
		{"?fields=id,*variants,variants.prices.amount", 200, `{"fields":["id","variants.*"],"limit":{"skip":0,"take":20}}`},
		{"?fields=id,variants.sku", 200, `{"fields":["id","variants.sku"],"limit":{"skip":0,"take":20}}`},
		// ordering by a field that isn't allowed fails
		{"?order=metadata", 400, `{"type":"invalid_data","message":"Order field metadata is not valid"}`},
	}
	for _, c := range cases {
		status, body, _ := do(t, "GET", srv.URL+"/store/custom"+c.query, "", nil)
		if status != c.status || body != c.want {
			t.Fatalf("%s: got %d %s, want %d %s", c.query, status, body, c.status, c.want)
		}
	}
}

func TestRelationsLimitAndDisallowed(t *testing.T) {
	cases := []struct {
		name   string
		config httputils.QueryConfig
		query  string
		status int
		want   string
	}{
		{"relations limit", ProductListConfigWithLimit, "?fields=*variants.options", 400, `{"type":"invalid_data","message":"The following fields expand more than the maximum of 1 allowed relations: variants.options.*"}`},
		{"within the limit", ProductListConfigWithLimit, "?fields=id,variants.title", 200, `{"fields":["id","variants.title"],"limit":{"skip":0,"take":20}}`},
		{"disallowed", ProductListConfigWithDisallowed, "?fields=id,variants.prices", 200, `{"fields":["id"],"limit":{"skip":0,"take":20}}`},
	}
	for _, c := range cases {
		route, mw := echoFields(c.config)
		srv := newServer(t, []medusahttp.RouteFile{route}, []medusahttp.MiddlewareRoute{mw})
		status, body, _ := do(t, "GET", srv.URL+"/store/custom"+c.query, "", nil)
		if status != c.status || body != c.want {
			t.Fatalf("%s: got %d %s, want %d %s", c.name, status, body, c.status, c.want)
		}
	}
}
