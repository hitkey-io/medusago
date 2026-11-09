package apikeys

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/api-key/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// AdminCreateApiKey mirrors AdminCreateApiKey.
var AdminCreateApiKey = zod.Object(
	zod.F("title", zod.String()),
	zod.F("type", zod.Enum("publishable", "secret")),
)

// AdminUpdateApiKey mirrors AdminUpdateApiKey.
var AdminUpdateApiKey = zod.Object(zod.F("title", zod.String()))

// AdminRevokeApiKey mirrors AdminRevokeApiKey.
var AdminRevokeApiKey = zod.Object(zod.F("revoke_in", zod.Number().Optional()))

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// WriteMiddlewares mirrors the write routes of adminApiKeyRoutesMiddlewares.
var WriteMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/api-keys", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateApiKey)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/api-keys/:id", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateApiKey)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/api-keys/:id/revoke", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminRevokeApiKey)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/api-keys/:id/sales-channels", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(utils.CreateLinkBody())), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
}

func init() { Middlewares = append(Middlewares, WriteMiddlewares...) }

// RefetchApiKey mirrors refetchApiKey.
func RefetchApiKey(req *medusahttp.MedusaRequest, id any, fields []string) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	return httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "api_key", f, fields)
}

// Respond refetches and writes { api_key } with status 200.
func Respond(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id any) error {
	v, err := RefetchApiKey(req, id, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if v != nil {
		out.Set("api_key", v)
	}
	return res.Status(200).JSON(out)
}

// Strings converts a validated string array.
func Strings(v any) []string {
	l, _ := v.([]any)
	if l == nil {
		return nil
	}
	out := make([]string, len(l))
	for i, x := range l {
		out[i] = fmt.Sprint(x)
	}
	return out
}

// POST mirrors POST: the created key (with its raw token) is returned
// without refetching.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	in := dal.NewRecord()
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		in.Set(k, v)
	}
	in.Set("created_by", req.AuthString("actor_id"))
	result, err := workflows.CreateApiKeysWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{in})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("api_key", result[0])
	return res.Status(200).JSON(out)
}
