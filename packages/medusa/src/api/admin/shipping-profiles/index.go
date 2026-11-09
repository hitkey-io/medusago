// Package shippingprofiles mirrors api/admin/shipping-profiles.
package shippingprofiles

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultAdminShippingProfileFields mirrors defaultAdminShippingProfileFields.
var DefaultAdminShippingProfileFields = []string{"id", "name", "type", "metadata", "created_at", "updated_at"}

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultAdminShippingProfileFields, Entity: "shipping_profile"}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultAdminShippingProfileFields, IsList: true, Entity: "shipping_profile"}
)

// AdminGetShippingProfilesParamsFields mirrors AdminGetShippingProfilesParamsFields.
var AdminGetShippingProfilesParamsFields = zod.Object(
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("q", zod.String().Optional()),
	zod.F("type", zod.String().Optional()),
	zod.F("name", zod.String().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetShippingProfilesParams mirrors AdminGetShippingProfilesParams.
var AdminGetShippingProfilesParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(20), Offset: utils.Num(0)}).
	Merge(AdminGetShippingProfilesParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetShippingProfilesParamsFields))

// AdminCreateShippingProfile mirrors AdminCreateShippingProfile.
var AdminCreateShippingProfile = zod.Object(
	zod.F("name", zod.String()),
	zod.F("type", zod.String()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
).Strict()

// AdminUpdateShippingProfile mirrors AdminUpdateShippingProfile.
var AdminUpdateShippingProfile = zod.Object(
	zod.F("name", zod.String().Optional()),
	zod.F("type", zod.String().Optional()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
).Strict()

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// Middlewares mirrors adminShippingProfilesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/shipping-profiles", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateShippingProfile)), httputils.ValidateAndTransformQuery(AdminGetShippingProfilesParams, RetrieveConfig))},
	{Methods: []string{"GET"}, Matcher: "/admin/shipping-profiles", Middlewares: mw(httputils.ValidateAndTransformQuery(AdminGetShippingProfilesParams, ListConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/shipping-profiles/:id", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateShippingProfile)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"GET"}, Matcher: "/admin/shipping-profiles/:id", Middlewares: mw(httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
}

// Respond refetches the profile and writes { shipping_profile }.
func Respond(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id any) error {
	f := dal.NewRecord()
	f.Set("id", id)
	v, err := httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "shipping_profile", f, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if v != nil {
		out.Set("shipping_profile", v)
	}
	return res.Status(200).JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	result, err := workflows.CreateShippingProfilesWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{body})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	return Respond(req, res, id)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("shipping_profiles", "shipping_profiles"), POST: POST}
