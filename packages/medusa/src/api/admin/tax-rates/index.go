// Package taxrates mirrors api/admin/tax-rates.
package taxrates

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/tax/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// Defaults mirrors defaults.
var Defaults = []string{"id", "name", "code", "rate", "tax_region_id", "is_default", "is_combinable", "created_by", "created_at", "updated_at", "deleted_at", "metadata", "*tax_region", "*rules"}

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: Defaults, Entity: "tax_rate"}
	ListConfig     = httputils.QueryConfig{Defaults: Defaults, IsList: true, Entity: "tax_rate"}
)

// AdminGetTaxRateParams mirrors AdminGetTaxRateParams.
var AdminGetTaxRateParams = utils.CreateSelectParams()

// AdminGetTaxRatesParamsFields mirrors AdminGetTaxRatesParamsFields.
var AdminGetTaxRatesParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("tax_region_id", zod.Union(zod.String(), zod.Array(zod.String()), utils.CreateOperatorMap(nil, nil)).Optional()),
	zod.F("is_default", zod.Union(zod.Literal("true"), zod.Literal("false")).Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetTaxRatesParams mirrors AdminGetTaxRatesParams.
var AdminGetTaxRatesParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(20), Offset: utils.Num(0)}).
	Merge(AdminGetTaxRatesParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetTaxRatesParamsFields))

// AdminCreateTaxRateRule mirrors AdminCreateTaxRateRule.
var AdminCreateTaxRateRule = zod.Object(
	zod.F("reference", zod.String()),
	zod.F("reference_id", zod.String()),
)

// AdminCreateTaxRate mirrors AdminCreateTaxRate.
var AdminCreateTaxRate = zod.Object(
	zod.F("rate", zod.Number().Optional()),
	zod.F("code", zod.String()),
	zod.F("rules", zod.Array(AdminCreateTaxRateRule).Optional()),
	zod.F("name", zod.String()),
	zod.F("is_default", zod.Boolean().Optional()),
	zod.F("is_combinable", zod.Boolean().Optional()),
	zod.F("tax_region_id", zod.String()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

// AdminUpdateTaxRate mirrors AdminUpdateTaxRate.
var AdminUpdateTaxRate = zod.Object(
	zod.F("rate", zod.Number().Optional()),
	zod.F("code", zod.String().Optional()),
	zod.F("rules", zod.Array(AdminCreateTaxRateRule).Optional()),
	zod.F("name", zod.String().Optional()),
	zod.F("is_default", zod.Boolean().Optional()),
	zod.F("is_combinable", zod.Boolean().Optional()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// Middlewares mirrors adminTaxRateRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/tax-rates", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateTaxRate)), httputils.ValidateAndTransformQuery(AdminGetTaxRateParams, RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/tax-rates/:id", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateTaxRate)), httputils.ValidateAndTransformQuery(AdminGetTaxRateParams, RetrieveConfig))},
	{Methods: []string{"GET"}, Matcher: "/admin/tax-rates/:id", Middlewares: mw(httputils.ValidateAndTransformQuery(AdminGetTaxRateParams, RetrieveConfig))},
	{Methods: []string{"GET"}, Matcher: "/admin/tax-rates", Middlewares: mw(httputils.ValidateAndTransformQuery(AdminGetTaxRatesParams, ListConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/tax-rates/:id/rules", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateTaxRateRule)), httputils.ValidateAndTransformQuery(AdminGetTaxRateParams, RetrieveConfig))},
	{Methods: []string{"DELETE"}, Matcher: "/admin/tax-rates/:id/rules/:rule_id", Middlewares: mw(httputils.ValidateAndTransformQuery(AdminGetTaxRateParams, RetrieveConfig))},
}

// RefetchTaxRate mirrors refetchTaxRate (nil when missing).
func RefetchTaxRate(req *medusahttp.MedusaRequest, id any, fields []string) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	return httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "tax_rate", f, fields)
}

// Respond writes { tax_rate } (undefined omitted).
func Respond(res *medusahttp.MedusaResponse, v any) error {
	out := dal.NewRecord()
	if v != nil {
		out.Set("tax_rate", v)
	}
	return res.Status(200).JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	in := dal.NewRecord()
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		in.Set(k, v)
	}
	in.Set("created_by", req.AuthString("actor_id"))
	result, err := workflows.CreateTaxRatesWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{in})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	v, err := RefetchTaxRate(req, id, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return Respond(res, v)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("tax_rate", "tax_rates"), POST: POST}
