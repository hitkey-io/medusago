// Package productcategories mirrors api/admin/product-categories.
package productcategories

import (
	categoryworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/product-category/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// Defaults mirrors defaults.
var Defaults = []string{"id", "name", "description", "handle", "is_active", "is_internal", "rank", "external_id", "parent_category_id", "created_at", "updated_at", "metadata", "*parent_category", "*category_children"}

// RetrieveConfig / ListConfig mirror the query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: Defaults, Entity: "product_category"}
	ListConfig     = httputils.QueryConfig{Defaults: Defaults, DefaultLimit: 50, IsList: true, Entity: "product_category"}
)

// AdminProductCategoryParams mirrors AdminProductCategoryParams.
var AdminProductCategoryParams = utils.CreateSelectParams().Merge(zod.Object(
	zod.F("include_ancestors_tree", commonvalidators.BooleanString().Optional()),
	zod.F("include_descendants_tree", commonvalidators.BooleanString().Optional()),
))

// AdminProductCategoriesParamsFields mirrors AdminProductCategoriesParamsFields.
var AdminProductCategoriesParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("description", utils.StringOrStrings().Optional()),
	zod.F("handle", utils.StringOrStrings().Optional()),
	zod.F("parent_category_id", utils.StringOrStrings().Optional()),
	zod.F("external_id", zod.Union(zod.String(), zod.Array(zod.String()), zod.Null()).Optional()),
	zod.F("include_ancestors_tree", commonvalidators.BooleanString().Optional()),
	zod.F("include_descendants_tree", commonvalidators.BooleanString().Optional()),
	zod.F("is_internal", commonvalidators.BooleanString().Optional()),
	zod.F("is_active", commonvalidators.BooleanString().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminProductCategoriesParams mirrors AdminProductCategoriesParams.
var AdminProductCategoriesParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(50), Offset: utils.Num(0)}).
	Merge(AdminProductCategoriesParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminProductCategoriesParamsFields))

// Middlewares mirrors adminProductCategoryRoutesMiddlewares (GET routes).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/product-categories", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminProductCategoriesParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/product-categories/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminProductCategoryParams, RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/product-categories", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateProductCategory)), httputils.ValidateAndTransformQuery(AdminProductCategoryParams, RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/product-categories/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateProductCategory)), httputils.ValidateAndTransformQuery(AdminProductCategoryParams, RetrieveConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/product-categories/:id/products", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(utils.CreateLinkBody())), httputils.ValidateAndTransformQuery(AdminProductCategoryParams, RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RefetchListRoute("product_category", "product_categories"), POST: POST}

// CreateProductCategory mirrors CreateProductCategory.
var CreateProductCategory = zod.Object(
	zod.F("name", zod.String()),
	zod.F("description", zod.String().Optional()),
	zod.F("handle", zod.String().Optional()),
	zod.F("is_internal", zod.Boolean().Optional()),
	zod.F("is_active", zod.Boolean().Optional()),
	zod.F("parent_category_id", zod.String().Nullish()),
	zod.F("external_id", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
	zod.F("rank", zod.Number().Nonnegative().Optional()),
).Strict()

// AdminCreateProductCategory mirrors AdminCreateProductCategory.
var AdminCreateProductCategory = utils.WithAdditionalData(CreateProductCategory)

// UpdateProductCategory mirrors UpdateProductCategory.
var UpdateProductCategory = zod.Object(
	zod.F("name", zod.String().Optional()),
	zod.F("description", zod.String().Optional()),
	zod.F("handle", zod.String().Optional()),
	zod.F("is_internal", zod.Boolean().Optional()),
	zod.F("is_active", zod.Boolean().Optional()),
	zod.F("parent_category_id", zod.String().Nullish()),
	zod.F("external_id", zod.String().Nullish()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Nullish()),
	zod.F("rank", zod.Number().Nonnegative().Optional()),
).Strict()

// AdminUpdateProductCategory mirrors AdminUpdateProductCategory.
var AdminUpdateProductCategory = utils.WithAdditionalData(UpdateProductCategory)

// SplitAdditional returns the body without additional_data, and it.
func SplitAdditional(body *dal.Record) (*dal.Record, *dal.Record) {
	rest := dal.NewRecord()
	var additional *dal.Record
	if body == nil {
		return rest, nil
	}
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		if k == "additional_data" {
			additional, _ = v.(*dal.Record)
			continue
		}
		rest.Set(k, v)
	}
	return rest, additional
}

// RefetchFirst mirrors refetchEntities({ entity: "product_category",
// idOrFilter: { id, ...filterableFields }, fields, pagination }).data[0].
func RefetchFirst(req *medusahttp.MedusaRequest, id any) (any, error) {
	filters := dal.NewRecord()
	filters.Set("id", id)
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			filters.Set(k, v)
		}
	}
	r, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{Entity: "product_category", IDOrFilter: filters, Fields: req.QueryConfig.Fields, Pagination: req.QueryConfig.Pagination})
	if err != nil {
		return nil, err
	}
	if len(r.Data) == 0 {
		return nil, nil
	}
	return r.Data[0], nil
}

// Respond writes { product_category } (undefined omitted).
func Respond(res *medusahttp.MedusaResponse, c any) error {
	out := dal.NewRecord()
	if c != nil {
		out.Set("product_category", c)
	}
	return res.Status(200).JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	rest, additional := SplitAdditional(body)
	result, err := categoryworkflows.CreateProductCategoriesWorkflow.Run(req.Ctx(), req.Scope, categoryworkflows.CreateProductCategoriesInput{ProductCategories: []*dal.Record{rest}, AdditionalData: additional})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	c, err := RefetchFirst(req, id)
	if err != nil {
		return err
	}
	return Respond(res, c)
}
