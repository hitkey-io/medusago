// Package returnreasons mirrors api/store/return-reasons.
package returnreasons

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// DefaultStoreRetrieveReturnReasonFields mirrors defaultStoreRetrieveReturnReasonFields.
var DefaultStoreRetrieveReturnReasonFields = []string{"id", "value", "label", "parent_return_reason_id", "description", "created_at", "updated_at", "deleted_at", "*parent_return_reason", "*return_reason_children"}

var allowed = storeutils.BuildAllowedFields(DefaultStoreRetrieveReturnReasonFields)

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultStoreRetrieveReturnReasonFields, Allowed: allowed}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultStoreRetrieveReturnReasonFields, Allowed: allowed, DefaultLimit: 20, IsList: true}
)

// Middlewares mirrors storeReturnReasonRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/store/return-reasons", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateFindParams(utils.FindParamsOptions{}), ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/store/return-reasons/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("return_reason", "return_reasons")}
