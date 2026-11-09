// Package pricepreferences mirrors api/admin/price-preferences.
package pricepreferences

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/pricing/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
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

// AdminCreatePricePreference mirrors AdminCreatePricePreference.
var AdminCreatePricePreference = zod.Object(
	zod.F("attribute", zod.String()),
	zod.F("value", zod.String()),
	zod.F("is_tax_inclusive", zod.Boolean().Optional()),
)

// AdminUpdatePricePreference mirrors AdminUpdatePricePreference.
var AdminUpdatePricePreference = zod.Object(
	zod.F("attribute", zod.String().Optional()),
	zod.F("value", zod.String().Optional()),
	zod.F("is_tax_inclusive", zod.Boolean().Optional()),
)

// Middlewares mirrors adminPricePreferencesRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/price-preferences", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetPricePreferencesParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/price-preferences/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/price-preferences", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminCreatePricePreference)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/price-preferences/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdatePricePreference)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RefetchListRoute("price_preference", "price_preferences"), POST: POST}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	result, err := workflows.CreatePricePreferencesWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{body})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	return Respond(req, res, id)
}

// Respond refetches the price preference and writes { price_preference }.
func Respond(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id any) error {
	entity, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "price_preference", id, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if entity != nil {
		out.Set("price_preference", entity)
	}
	return res.Status(200).JSON(out)
}
