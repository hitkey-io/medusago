package customergroups

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer-group/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// AdminCreateCustomerGroup mirrors AdminCreateCustomerGroup.
var AdminCreateCustomerGroup = zod.Object(
	zod.F("name", zod.String()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

// AdminUpdateCustomerGroup mirrors AdminUpdateCustomerGroup.
var AdminUpdateCustomerGroup = zod.Object(
	zod.F("name", zod.String().Optional()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
)

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// WriteMiddlewares mirrors the write routes of adminCustomerGroupRoutesMiddlewares.
var WriteMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/customer-groups", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateCustomerGroup)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/customer-groups/:id", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateCustomerGroup)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/customer-groups/:id/customers", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(utils.CreateLinkBody())), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
}

func init() { Middlewares = append(Middlewares, WriteMiddlewares...) }

// RefetchCustomerGroup mirrors refetchCustomerGroup (nil when missing).
func RefetchCustomerGroup(req *medusahttp.MedusaRequest, id any, fields []string) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	return httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "customer_group", f, fields)
}

// Respond refetches the group and writes { customer_group }.
func Respond(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id any) error {
	v, err := RefetchCustomerGroup(req, id, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if v != nil {
		out.Set("customer_group", v)
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
	result, err := workflows.CreateCustomerGroupsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{in})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	return Respond(req, res, id)
}
