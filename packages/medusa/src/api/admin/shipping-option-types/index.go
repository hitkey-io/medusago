// Package shippingoptiontypes mirrors api/admin/shipping-option-types.
package shippingoptiontypes

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/shipping-options/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// Defaults mirrors defaultAdminShippingOptionTypeFields.
var Defaults = []string{"id", "label", "code", "description", "created_at", "updated_at"}

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: Defaults, Entity: "shipping_option_type"}
	ListConfig     = httputils.QueryConfig{Defaults: Defaults, DefaultLimit: 20, IsList: true, Entity: "shipping_option_type"}
)

// AdminGetShippingOptionTypesParamsFields mirrors AdminGetShippingOptionTypesParamsFields.
var AdminGetShippingOptionTypesParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("label", utils.StringOrStrings().Optional()),
	zod.F("code", utils.StringOrStrings().Optional()),
	zod.F("description", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetShippingOptionTypesParams mirrors AdminGetShippingOptionTypesParams.
var AdminGetShippingOptionTypesParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(10), Offset: utils.Num(0)}).
	Merge(AdminGetShippingOptionTypesParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetShippingOptionTypesParamsFields))

// AdminCreateShippingOptionType mirrors AdminCreateShippingOptionType.
var AdminCreateShippingOptionType = zod.Object(
	zod.F("label", zod.String()),
	zod.F("code", zod.String()),
	zod.F("description", zod.String().Optional()),
).Strict()

// AdminUpdateShippingOptionType mirrors AdminUpdateShippingOptionType.
var AdminUpdateShippingOptionType = zod.Object(
	zod.F("label", zod.String().Optional()),
	zod.F("code", zod.String().Optional()),
	zod.F("description", zod.String().Optional()),
).Strict()

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// Middlewares mirrors adminShippingOptionTypeRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/shipping-option-types", Middlewares: mw(httputils.ValidateAndTransformQuery(AdminGetShippingOptionTypesParams, ListConfig))},
	{Methods: []string{"GET"}, Matcher: "/admin/shipping-option-types/:id", Middlewares: mw(httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/shipping-option-types", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateShippingOptionType)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/shipping-option-types/:id", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateShippingOptionType)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
}

// Refetch mirrors refetchShippingOptionType.
func Refetch(req *medusahttp.MedusaRequest, id any, fields []string) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	r, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{Entity: "shipping_option_type", IDOrFilter: f, Fields: fields})
	if err != nil || len(r.Data) == 0 {
		return nil, err
	}
	return r.Data[0], nil
}

// Respond writes { shipping_option_type } with status 200.
func Respond(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id any) error {
	v, err := Refetch(req, id, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if v != nil {
		out.Set("shipping_option_type", v)
	}
	return res.Status(200).JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	result, err := workflows.CreateShippingOptionTypesWorkflow.Run(req.Ctx(), req.Scope, workflows.CreateShippingOptionTypesInput{ShippingOptionTypes: []*dal.Record{body}})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	return Respond(req, res, id)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RefetchListRoute("shipping_option_type", "shipping_option_types"), POST: POST}
