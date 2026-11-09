package http

import (
	"log/slog"
	"regexp"
	"strings"
)

// HTTPMethods mirrors HTTP_METHODS.
var HTTPMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"}

// Route is the Go form of a route.ts file: its exported handlers and flags.
type Route struct {
	GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD RouteHandler
	// Authenticate mirrors `export const AUTHENTICATE = false` (nil = true).
	Authenticate *bool
	// Cors mirrors `export const CORS = false` (nil = true).
	Cors *bool
}

// RouteFile is a route file at its URL path (e.g. "/admin/users/:id").
type RouteFile struct {
	Path  string
	Route Route
}

// RouteDescriptor mirrors RouteDescriptor.
type RouteDescriptor struct {
	Matcher               string
	Method                string
	Handler               RouteHandler
	OptedOutOfAuth        bool
	ShouldAppendAdminCors bool
	ShouldAppendAuthCors  bool
	ShouldAppendStoreCors bool
}

func (r RouteDescriptor) sortMatcher() (string, bool)           { return r.Matcher, false }
func (r RouteDescriptor) sortHasMethod() bool                   { return true }
func (r RouteDescriptor) findMatcher() (string, *regexp.Regexp) { return r.Matcher, nil }
func (r RouteDescriptor) findMethods() []string                 { return []string{r.Method} }

// BodyParserConfig mirrors ParserConfigArgs; Disabled mirrors `false`.
type BodyParserConfig struct {
	Disabled        bool
	SizeLimit       string
	PreserveRawBody bool
}

// MiddlewareRoute mirrors MiddlewareRoute of defineMiddlewares.
type MiddlewareRoute struct {
	Matcher   string
	MatcherRe *regexp.Regexp
	// Methods (nil = all, registered with app.use).
	Methods     []string
	Middlewares []MiddlewareFunction
	BodyParser  *BodyParserConfig
}

// MiddlewareDescriptor mirrors MiddlewareDescriptor.
type MiddlewareDescriptor struct {
	Matcher   string
	MatcherRe *regexp.Regexp
	Methods   []string
	Handler   MiddlewareFunction
}

func (m MiddlewareDescriptor) sortMatcher() (string, bool) {
	if m.MatcherRe != nil {
		return m.MatcherRe.String(), true
	}
	return m.Matcher, false
}
func (m MiddlewareDescriptor) sortHasMethod() bool { return m.Methods != nil }

// BodyParserConfigRoute mirrors BodyParserConfigRoute.
type BodyParserConfigRoute struct {
	Matcher   string
	MatcherRe *regexp.Regexp
	Methods   []string
	Config    BodyParserConfig
}

func (b BodyParserConfigRoute) sortMatcher() (string, bool) {
	if b.MatcherRe != nil {
		return b.MatcherRe.String(), true
	}
	return b.Matcher, false
}
func (b BodyParserConfigRoute) sortHasMethod() bool                   { return true }
func (b BodyParserConfigRoute) findMatcher() (string, *regexp.Regexp) { return b.Matcher, b.MatcherRe }
func (b BodyParserConfigRoute) findMethods() []string                 { return b.Methods }

// ApiLoaderOptions configures the ApiLoader.
type ApiLoaderOptions struct {
	Routes      []RouteFile
	Middlewares []MiddlewareRoute
	// ErrorHandler replaces the core error handler when set.
	ErrorHandler ErrorHandlerFunction
	AdminCors    string
	StoreCors    string
	AuthCors     string
	// Authenticate builds the authenticate middleware (avoids an import
	// cycle with the middlewares package).
	Authenticate func(actorType []string, authType []string, opts AuthenticateOptions) MiddlewareFunction
	// BodyParser builds the json/text/urlencoded parsers for a config.
	BodyParser func(kind string, cfg *BodyParserConfig) MiddlewareFunction
	// StorePublishableKey and StoreLocale mirror the /store middlewares.
	StorePublishableKey  MiddlewareFunction
	StoreLocale          MiddlewareFunction
	BaseRestrictedFields []string
	StoreRelationsLimit  int
	CoreErrorHandler     ErrorHandlerFunction
}

// AuthenticateOptions mirrors the options of authenticate().
type AuthenticateOptions struct {
	AllowUnauthenticated bool
	AllowUnregistered    bool
	// RequireMfa: nil = default (!AllowUnauthenticated).
	RequireMfa *bool
	// RequireMfaMaxAgeSeconds mirrors requireMfa: { maxAgeSeconds }.
	RequireMfaMaxAgeSeconds int
}

func routeType(path string) string {
	switch {
	case strings.HasPrefix(path, "/admin"):
		return "admin"
	case strings.HasPrefix(path, "/rbac"):
		return "rbac"
	case strings.HasPrefix(path, "/store"):
		return "store"
	case strings.HasPrefix(path, "/auth"):
		return "auth"
	}
	return ""
}

// routeDescriptors mirrors RoutesLoader.#getRoutesForFile.
func routeDescriptors(f RouteFile) []RouteDescriptor {
	shouldAuthenticate := f.Route.Authenticate == nil || *f.Route.Authenticate
	shouldApplyCors := f.Route.Cors == nil || *f.Route.Cors
	rt := routeType(f.Path)
	handlers := []struct {
		m string
		h RouteHandler
	}{{"GET", f.Route.GET}, {"POST", f.Route.POST}, {"PUT", f.Route.PUT}, {"PATCH", f.Route.PATCH}, {"DELETE", f.Route.DELETE}, {"OPTIONS", f.Route.OPTIONS}, {"HEAD", f.Route.HEAD}}
	var out []RouteDescriptor
	for _, x := range handlers {
		if x.h == nil {
			continue
		}
		out = append(out, RouteDescriptor{
			Matcher:               f.Path,
			Method:                x.m,
			Handler:               x.h,
			OptedOutOfAuth:        !shouldAuthenticate,
			ShouldAppendAdminCors: shouldApplyCors && rt == "admin",
			ShouldAppendAuthCors:  shouldApplyCors && rt == "auth",
			ShouldAppendStoreCors: shouldApplyCors && rt == "store",
		})
	}
	return out
}

// wrapHandler mirrors wrapHandler for routes.
func wrapRoute(h RouteHandler) MiddlewareFunction {
	return func(req *MedusaRequest, res *MedusaResponse, next NextFunction) error {
		if len(req.Errors) > 0 {
			return res.Status(400).JSON(errorsBody(req.Errors))
		}
		return h(req, res)
	}
}

// wrapMiddleware mirrors wrapHandler for middlewares.
func wrapMiddleware(h MiddlewareFunction) MiddlewareFunction {
	return func(req *MedusaRequest, res *MedusaResponse, next NextFunction) error {
		if len(req.Errors) > 0 {
			return res.Status(400).JSON(errorsBody(req.Errors))
		}
		return h(req, res, next)
	}
}

// LoadAPI mirrors ApiLoader.load.
func LoadAPI(app *App, o ApiLoaderOptions) {
	// #assignRestrictedFields
	app.Use("/", func(req *MedusaRequest, res *MedusaResponse, next NextFunction) error {
		req.Allowed = []string{}
		next(nil)
		return nil
	})
	app.Use("/store", func(req *MedusaRequest, res *MedusaResponse, next NextFunction) error {
		req.RestrictedFields = append([]string{}, o.BaseRestrictedFields...)
		req.StoreRelationsLimit = o.StoreRelationsLimit
		next(nil)
		return nil
	})
	app.Use("/admin", func(req *MedusaRequest, res *MedusaResponse, next NextFunction) error {
		req.RestrictedFields = []string{}
		next(nil)
		return nil
	})

	var routes []RouteDescriptor
	for _, f := range o.Routes {
		routes = append(routes, routeDescriptors(f)...)
	}
	var middlewares []MiddlewareDescriptor
	var bodyParserRoutes []BodyParserConfigRoute
	for _, mr := range o.Middlewares {
		if mr.BodyParser != nil {
			methods := mr.Methods
			if methods == nil {
				methods = HTTPMethods
			}
			bodyParserRoutes = append(bodyParserRoutes, BodyParserConfigRoute{Matcher: mr.Matcher, MatcherRe: mr.MatcherRe, Methods: methods, Config: *mr.BodyParser})
		}
		for _, mw := range mr.Middlewares {
			middlewares = append(middlewares, MiddlewareDescriptor{Matcher: mr.Matcher, MatcherRe: mr.MatcherRe, Methods: mr.Methods, Handler: mw})
		}
	}
	routesFinder := NewRoutesFinder[RouteDescriptor](nil)

	// Body parsers
	sortable := make([]SortableRoute, len(bodyParserRoutes))
	for i, r := range bodyParserRoutes {
		sortable[i] = r
	}
	var sortedBodyParsers []BodyParserConfigRoute
	for _, r := range SortRoutes(sortable, []string{"static", "params", "regex", "wildcard", "global"}) {
		sortedBodyParsers = append(sortedBodyParsers, r.(BodyParserConfigRoute))
	}
	bodyParserFinder := NewRoutesFinder(sortedBodyParsers)
	if o.BodyParser != nil {
		for _, kind := range []string{"json", "text", "urlencoded"} {
			kind := kind
			app.Use("/", func(req *MedusaRequest, res *MedusaResponse, next NextFunction) error {
				m, ok := bodyParserFinder.Find(req.Path(), req.Method)
				if !ok {
					return o.BodyParser(kind, nil)(req, res, next)
				}
				if m.Config.Disabled {
					next(nil)
					return nil
				}
				cfg := m.Config
				return o.BodyParser(kind, &cfg)(req, res, next)
			})
		}
	}

	applyCors := func(namespace string, cors string, pick func(RouteDescriptor) bool) {
		fn := Cors(ParseCorsOrigins(cors))
		app.Use(namespace, func(req *MedusaRequest, res *MedusaResponse, next NextFunction) error {
			method := req.Method
			if method == "OPTIONS" {
				if m := req.Header.Get("Access-Control-Request-Method"); m != "" {
					method = m
				}
			}
			if r, ok := routesFinder.Find(req.Path(), method); ok && pick(r) {
				return fn(req, res, next)
			}
			next(nil)
			return nil
		})
	}
	applyAuth := func(namespace string, actorTypes, authTypes []string, opts AuthenticateOptions) {
		if o.Authenticate == nil {
			return
		}
		fn := o.Authenticate(actorTypes, authTypes, opts)
		app.Use(namespace, func(req *MedusaRequest, res *MedusaResponse, next NextFunction) error {
			if r, ok := routesFinder.Find(req.Path(), req.Method); ok && r.OptedOutOfAuth {
				next(nil)
				return nil
			}
			return fn(req, res, next)
		})
	}

	applyCors("/admin", o.AdminCors, func(r RouteDescriptor) bool { return r.ShouldAppendAdminCors })
	applyAuth("/admin", []string{"user"}, []string{"bearer", "session", "api-key"}, AuthenticateOptions{})
	applyCors("/store", o.StoreCors, func(r RouteDescriptor) bool { return r.ShouldAppendStoreCors })
	if o.StorePublishableKey != nil {
		app.Use("/store", o.StorePublishableKey)
	}
	if o.StoreLocale != nil {
		app.Use("/store", o.StoreLocale)
	}
	applyAuth("/store", []string{"customer"}, []string{"bearer", "session"}, AuthenticateOptions{AllowUnauthenticated: true})
	applyCors("/auth", o.AuthCors, func(r RouteDescriptor) bool { return r.ShouldAppendAuthCors })

	var collection []SortableRoute
	for _, m := range middlewares {
		collection = append(collection, m)
	}
	for _, r := range routes {
		collection = append(collection, r)
	}
	for _, item := range SortRoutes(collection, nil) {
		switch r := item.(type) {
		case RouteDescriptor:
			routesFinder.Add(r)
			slog.Debug("registering route " + r.Method + " " + r.Matcher)
			app.Handle(r.Method, r.Matcher, wrapRoute(r.Handler))
		case MiddlewareDescriptor:
			if r.Methods == nil {
				if r.MatcherRe != nil {
					app.UseRegexp(r.MatcherRe, wrapMiddleware(r.Handler))
				} else {
					app.Use(r.Matcher, wrapMiddleware(r.Handler))
				}
				continue
			}
			for _, m := range r.Methods {
				if r.MatcherRe != nil {
					app.HandleRegexp(m, r.MatcherRe, wrapMiddleware(r.Handler))
				} else {
					app.Handle(m, r.Matcher, wrapMiddleware(r.Handler))
				}
			}
		}
	}

	if o.ErrorHandler != nil {
		app.UseError(o.ErrorHandler)
	} else if o.CoreErrorHandler != nil {
		app.UseError(o.CoreErrorHandler)
	}
}

func errorsBody(errs []any) map[string]any {
	return map[string]any{
		"errors":  errs,
		"message": "Provided request body contains errors. Please check the data and retry the request",
	}
}
