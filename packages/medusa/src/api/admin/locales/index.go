// Package locales mirrors api/admin/locales (route.ts, middlewares.ts,
// validators.ts and query-config.ts).
package locales

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultAdminLocaleFields mirrors defaultAdminLocaleFields.
var DefaultAdminLocaleFields = []string{"code", "name"}

// Query configs.
var (
	RetrieveTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminLocaleFields, Entity: "store_locale"}
	ListTransformQueryConfig     = httputils.QueryConfig{Defaults: DefaultAdminLocaleFields, DefaultLimit: 200, IsList: true, Entity: "store_locale"}
)

// AdminGetLocalesParamsFields mirrors AdminGetLocalesParamsFields.
var AdminGetLocalesParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("code", utils.StringOrStrings().Optional()),
)

// AdminGetLocalesParams mirrors AdminGetLocalesParams.
var AdminGetLocalesParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(200)}).
	Merge(AdminGetLocalesParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetLocalesParamsFields))

// Middlewares mirrors adminLocalesRoutesMiddlewares (the authorize guards
// are RBAC, not ported).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/locales", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetLocalesParams, ListTransformQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/locales/:code", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveTransformQueryConfig)}},
}

// Route is route.ts: GET lists the locales through query.graph.
var Route = medusahttp.Route{GET: httputils.RefetchListRoute("locale", "locales")}
