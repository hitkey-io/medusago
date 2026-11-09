// Package taxproviders mirrors api/admin/tax-providers.
package taxproviders

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// ListConfig mirrors listTransformQueryConfig.
var ListConfig = httputils.QueryConfig{Defaults: []string{"id", "is_enabled"}, IsList: true, Entity: "tax_provider"}

// AdminGetTaxProvidersParamsFields mirrors AdminGetTaxProvidersParamsFields.
var AdminGetTaxProvidersParamsFields = zod.Object(
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("is_enabled", zod.Boolean().Optional()),
)

// AdminGetTaxProvidersParams mirrors AdminGetTaxProvidersParams.
var AdminGetTaxProvidersParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(20), Offset: utils.Num(0)}).
	Merge(AdminGetTaxProvidersParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetTaxProvidersParamsFields))

// Middlewares mirrors adminTaxProviderRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/tax-providers", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetTaxProvidersParams, ListConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RefetchListRoute("tax_providers", "tax_providers")}
