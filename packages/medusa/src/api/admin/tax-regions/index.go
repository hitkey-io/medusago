// Package taxregions mirrors api/admin/tax-regions.
package taxregions

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// Defaults mirrors defaults.
var Defaults = []string{"id", "country_code", "province_code", "parent_id", "provider_id", "created_by", "created_at", "updated_at", "deleted_at", "metadata", "*children", "*children.tax_rates", "*children.tax_rates.rules", "*parent", "*tax_rates", "*tax_rates.rules"}

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: Defaults, Entity: "tax_region"}
	ListConfig     = httputils.QueryConfig{Defaults: Defaults, IsList: true, Entity: "tax_region"}
)

func stringsOrOperators() *zod.Schema {
	return zod.Union(zod.String(), zod.Array(zod.String()), utils.CreateOperatorMap(nil, nil))
}

// AdminGetTaxRegionsParamsFields mirrors AdminGetTaxRegionsParamsFields.
var AdminGetTaxRegionsParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("country_code", stringsOrOperators().Optional()),
	zod.F("province_code", stringsOrOperators().Optional()),
	zod.F("parent_id", stringsOrOperators().Optional()),
	zod.F("created_by", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetTaxRegionsParams mirrors AdminGetTaxRegionsParams.
var AdminGetTaxRegionsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(20), Offset: utils.Num(0)}).
	Merge(AdminGetTaxRegionsParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetTaxRegionsParamsFields))

// Middlewares mirrors adminTaxRegionRoutesMiddlewares (reads).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/tax-regions", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetTaxRegionsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/tax-regions/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("tax_regions", "tax_regions")}
