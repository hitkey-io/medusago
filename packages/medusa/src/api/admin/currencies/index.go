// Package currencies mirrors api/admin/currencies.
package currencies

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultAdminCurrencyFields mirrors defaultAdminCurrencyFields.
var DefaultAdminCurrencyFields = []string{"code", "name", "symbol", "symbol_native", "decimal_digits", "rounding"}

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultAdminCurrencyFields, Entity: "currency"}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultAdminCurrencyFields, DefaultLimit: 200, IsList: true, Entity: "currency"}
)

// AdminGetCurrenciesParamsFields mirrors AdminGetCurrenciesParamsFields.
var AdminGetCurrenciesParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("code", utils.StringOrStrings().Optional()),
)

// AdminGetCurrenciesParams mirrors AdminGetCurrenciesParams.
var AdminGetCurrenciesParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(200)}).
	Merge(AdminGetCurrenciesParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetCurrenciesParamsFields))

// Middlewares mirrors adminCurrencyRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/currencies", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetCurrenciesParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/currencies/:code", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("currency", "currencies")}
