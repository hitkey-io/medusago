// Package productcategories mirrors api/admin/product-categories.
package productcategories

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
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
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RefetchListRoute("product_category", "product_categories")}
