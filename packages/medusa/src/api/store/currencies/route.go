// Package currencies mirrors api/store/currencies.
package currencies

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultStoreCurrencyFields mirrors defaultStoreCurrencyFields.
var DefaultStoreCurrencyFields = []string{"code", "name", "symbol", "symbol_native", "decimal_digits", "rounding"}

var allowed = storeutils.BuildAllowedFields(DefaultStoreCurrencyFields, []string{"created_at", "updated_at", "deleted_at"})

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultStoreCurrencyFields, Allowed: allowed}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultStoreCurrencyFields, Allowed: allowed, DefaultLimit: 50, IsList: true}
)

// StoreGetCurrenciesParamsFields mirrors StoreGetCurrenciesParamsFields.
var StoreGetCurrenciesParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("code", utils.StringOrStrings().Optional()),
)

// StoreGetCurrenciesParams mirrors StoreGetCurrenciesParams.
var StoreGetCurrenciesParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(50)}).
	Merge(StoreGetCurrenciesParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(StoreGetCurrenciesParamsFields))

// Middlewares mirrors storeCurrencyRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/store/currencies", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(StoreGetCurrenciesParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/store/currencies/:code", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("currency", "currencies")}
