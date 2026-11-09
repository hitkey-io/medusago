// Package taxregions mirrors api/admin/tax-regions.
package taxregions

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/tax/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
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

// CreateTaxRegion mirrors CreateTaxRegion.
var CreateTaxRegion = zod.Object(
	zod.F("country_code", zod.String()),
	zod.F("provider_id", zod.String().Nullish()),
	zod.F("province_code", zod.String().Nullish()),
	zod.F("parent_id", zod.String().Nullish()),
	zod.F("default_tax_rate", zod.Object(
		zod.F("rate", zod.Number().Optional()),
		zod.F("code", zod.String()),
		zod.F("name", zod.String()),
		zod.F("is_combinable", zod.Boolean().Optional()),
		zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
	).Optional()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	}
	return true
}

// AdminCreateTaxRegion mirrors AdminCreateTaxRegion (WithAdditionalData +
// refine on provider_id).
var AdminCreateTaxRegion = utils.WithAdditionalData(CreateTaxRegion).SuperRefine(func(v any, add func(*zod.Issue)) {
	r, ok := v.(*dal.Record)
	if !ok {
		return
	}
	p, _ := r.Get("parent_id")
	pr, _ := r.Get("provider_id")
	if !truthy(p) && !truthy(pr) {
		add(&zod.Issue{Path: []any{"provider_id"}, Message: "Provider is required when creating a non-province tax region."})
	}
})

// AdminUpdateTaxRegion mirrors AdminUpdateTaxRegion.
var AdminUpdateTaxRegion = zod.Object(
	zod.F("province_code", zod.String().Nullish()),
	zod.F("provider_id", zod.String().Optional()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

// Middlewares mirrors adminTaxRegionRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/tax-regions", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateTaxRegion)), httputils.ValidateAndTransformQuery(AdminGetTaxRegionsParams, RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/tax-regions/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateTaxRegion)), httputils.ValidateAndTransformQuery(AdminGetTaxRegionsParams, RetrieveConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/tax-regions", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetTaxRegionsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/tax-regions/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("tax_regions", "tax_regions"), POST: POST}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	in := dal.NewRecord()
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		in.Set(k, v)
	}
	in.Set("created_by", req.AuthString("actor_id"))
	result, err := workflows.CreateTaxRegionsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{in})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	return Respond(req, res, id)
}

// Respond refetches the tax region and writes { tax_region }.
func Respond(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id any) error {
	f := dal.NewRecord()
	f.Set("id", id)
	v, err := httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "tax_region", f, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if v != nil {
		out.Set("tax_region", v)
	}
	return res.Status(200).JSON(out)
}

// RespondGraph mirrors the query.graph refetch with throwIfKeyNotFound.
func RespondGraph(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id any) error {
	f := dal.NewRecord()
	f.Set("id", id)
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: "tax_region", Fields: req.QueryConfig.Fields, Filters: f}, joiner.Options{ThrowIfKeyNotFound: true})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if len(r.Data) > 0 {
		out.Set("tax_region", r.Data[0])
	}
	return res.JSON(out)
}
