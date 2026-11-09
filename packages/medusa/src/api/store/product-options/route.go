// Package productoptions mirrors api/store/product-options.
package productoptions

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// Defaults mirrors defaults.
var Defaults = []string{"id", "title", "is_exclusive", "values.*", "created_at", "updated_at", "metadata"}

var allowed = storeutils.BuildAllowedFields(Defaults, []string{"values.id", "values.value"})

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: Defaults, Allowed: allowed}
	ListConfig     = httputils.QueryConfig{Defaults: Defaults, Allowed: allowed, DefaultLimit: 50, IsList: true}
)

// ParamsFields mirrors StoreProductOptionsParamsFields.
var ParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("title", utils.StringOrStrings().Optional()),
	zod.F("is_exclusive", commonvalidators.BooleanString().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// ListParams mirrors StoreProductOptionsParams.
var ListParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(50)}).
	Merge(ParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(ParamsFields))

// Middlewares mirrors storeProductOptionRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/store/product-options", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(ListParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/store/product-options/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams().Merge(zod.Object()), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: storeutils.GraphListRoute("product_option", "product_options", false, true)}
