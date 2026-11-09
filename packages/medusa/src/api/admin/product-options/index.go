// Package productoptions mirrors api/admin/product-options.
package productoptions

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultAdminProductOptionsFields mirrors defaultAdminProductOptionsFields.
var DefaultAdminProductOptionsFields = []string{"id", "title", "is_exclusive", "values.*", "products.*", "created_at", "updated_at", "metadata"}

// RetrieveConfig / ListConfig mirror the query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultAdminProductOptionsFields, Entity: "product_option"}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultAdminProductOptionsFields, DefaultLimit: 20, IsList: true, Entity: "product_option"}
)

// AdminGetProductOptionsParamsFields mirrors AdminGetProductOptionsParamsFields.
var AdminGetProductOptionsParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("title", utils.StringOrStrings().Optional()),
	zod.F("is_exclusive", commonvalidators.BooleanString().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetProductOptionsParams mirrors AdminGetProductOptionsParams.
var AdminGetProductOptionsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(20), Offset: utils.Num(0)}).
	Merge(AdminGetProductOptionsParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetProductOptionsParamsFields))

// Middlewares mirrors adminProductOptionRoutesMiddlewares (authorize
// entries are Enterprise and skipped).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/product-options", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetProductOptionsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/product-options/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/product-options", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateProductOption)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/product-options/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateProductOption)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RefetchListRoute("product_option", "product_options"), POST: POST}

// AdminCreateProductOption mirrors AdminCreateProductOption.
var AdminCreateProductOption = zod.Object(
	zod.F("title", zod.String()),
	zod.F("values", zod.Array(zod.String())),
	zod.F("ranks", zod.Record(zod.String(), zod.Number()).Optional()),
	zod.F("is_exclusive", zod.Boolean().Optional()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
).Strict()

// AdminUpdateProductOption mirrors AdminUpdateProductOption.
var AdminUpdateProductOption = zod.Object(
	zod.F("title", zod.String().Optional()),
	zod.F("values", zod.Array(zod.String()).Optional()),
	zod.F("ranks", zod.Record(zod.String(), zod.Number()).Optional()),
	zod.F("is_exclusive", zod.Boolean().Optional()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
).Strict()

// GraphOne mirrors query.graph({ entity: "product_option", filters, fields })
// and returns data[0] (nil when missing).
func GraphOne(req *medusahttp.MedusaRequest, entity string, filters *dal.Record, fields []string) (any, error) {
	r, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{Entity: entity, IDOrFilter: filters, Fields: fields})
	if err != nil {
		return nil, err
	}
	if len(r.Data) == 0 {
		return nil, nil
	}
	return r.Data[0], nil
}

// Respond writes { <key>: entity } (undefined omitted).
func Respond(res *medusahttp.MedusaResponse, key string, entity any) error {
	out := dal.NewRecord()
	if entity != nil {
		out.Set(key, entity)
	}
	return res.Status(200).JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	result, err := workflows.CreateProductOptionsWorkflow.Run(req.Ctx(), req.Scope, workflows.CreateProductOptionsInput{ProductOptions: []*dal.Record{body}})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	f := dal.NewRecord()
	f.Set("id", id)
	o, err := GraphOne(req, "product_option", f, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return Respond(res, "product_option", o)
}
