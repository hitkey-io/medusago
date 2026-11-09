// Package pricepreferences mirrors api/admin/price-preferences.
package pricepreferences

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// AdminPricePreferenceRemoteQueryFields mirrors adminPricePreferenceRemoteQueryFields.
var AdminPricePreferenceRemoteQueryFields = []string{"id", "attribute", "value", "is_tax_inclusive", "created_at", "deleted_at", "updated_at"}

// RetrieveConfig / ListConfig mirror the query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: AdminPricePreferenceRemoteQueryFields, Entity: "price_preference"}
	ListConfig     = httputils.QueryConfig{Defaults: AdminPricePreferenceRemoteQueryFields, IsList: true, Entity: "price_preference"}
)

// AdminGetPricePreferencesParamsFields mirrors AdminGetPricePreferencesParamsFields.
var AdminGetPricePreferencesParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("attribute", utils.StringOrStrings().Optional()),
	zod.F("value", utils.StringOrStrings().Optional()),
)

// AdminGetPricePreferencesParams mirrors AdminGetPricePreferencesParams.
var AdminGetPricePreferencesParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(300), Offset: utils.Num(0)}).
	Merge(AdminGetPricePreferencesParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetPricePreferencesParamsFields))

// Middlewares mirrors adminPricePreferencesRoutesMiddlewares (GET routes).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/price-preferences", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetPricePreferencesParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/price-preferences/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RefetchListRoute("price_preference", "price_preferences")}
