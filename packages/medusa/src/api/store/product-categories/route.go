// Package productcategories mirrors api/store/product-categories.
package productcategories

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// Defaults mirrors defaults.
var Defaults = []string{"id", "name", "description", "handle", "rank", "external_id", "parent_category_id", "created_at", "updated_at", "metadata", "*parent_category", "*category_children"}

var storeCategoryFields = []string{"id", "name", "description", "handle", "rank", "external_id", "parent_category_id", "created_at", "updated_at", "deleted_at"}

var storeCategoryRelations = []string{"parent_category", "category_children"}

// buildNestedCategoryFields mirrors buildNestedCategoryFields.
func buildNestedCategoryFields(depth int) []string {
	var paths []string
	prefixes := storeCategoryRelations
	for level := 1; level <= depth; level++ {
		if level > 1 {
			paths = append(paths, prefixes...)
		}
		for _, p := range prefixes {
			paths = append(paths, storeutils.PrefixAllowedFields(p, storeCategoryFields)...)
		}
		var next []string
		for _, p := range prefixes {
			next = append(next, storeutils.PrefixAllowedFields(p, storeCategoryRelations)...)
		}
		prefixes = next
	}
	return paths
}

var allowed = storeutils.BuildAllowedFields(Defaults, append([]string{"products", "deleted_at", "products.title", "products.variants", "products.options", "products.images", "category_children.products.title"}, buildNestedCategoryFields(3)...))

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: Defaults, Allowed: allowed}
	ListConfig     = httputils.QueryConfig{Defaults: Defaults, Allowed: allowed, DefaultLimit: 50, IsList: true}
)

// StoreProductCategoryParams mirrors StoreProductCategoryParams.
var StoreProductCategoryParams = utils.CreateSelectParams().Merge(zod.Object(
	zod.F("include_ancestors_tree", commonvalidators.BooleanString().Optional()),
	zod.F("include_descendants_tree", commonvalidators.BooleanString().Optional()),
))

// StoreProductCategoriesParamsFields mirrors StoreProductCategoriesParamsFields.
var StoreProductCategoriesParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("name", utils.StringOrStrings().Optional()),
	zod.F("description", utils.StringOrStrings().Optional()),
	zod.F("handle", utils.StringOrStrings().Optional()),
	zod.F("parent_category_id", utils.StringOrStrings().Optional()),
	zod.F("external_id", zod.Union(zod.String(), zod.Array(zod.String()), zod.Null()).Optional()),
	zod.F("include_ancestors_tree", commonvalidators.BooleanString().Optional()),
	zod.F("include_descendants_tree", commonvalidators.BooleanString().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// StoreProductCategoriesParams mirrors StoreProductCategoriesParams.
var StoreProductCategoriesParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(50)}).
	Merge(StoreProductCategoriesParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(StoreProductCategoriesParamsFields))

// ApplyCategoryFilters mirrors applyCategoryFilters.
func ApplyCategoryFilters(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
	f := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			f.Set(k, v)
		}
	}
	f.Set("is_active", true)
	f.Set("is_internal", false)
	req.FilterableFields = f
	next(nil)
	return nil
}

// Middlewares mirrors storeProductCategoryRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/store/product-categories", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(StoreProductCategoriesParams, ListConfig), ApplyCategoryFilters}},
	{Methods: []string{"GET"}, Matcher: "/store/product-categories/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(StoreProductCategoryParams, RetrieveConfig), ApplyCategoryFilters}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: storeutils.GraphListRoute("product_category", "product_categories", true, false)}
