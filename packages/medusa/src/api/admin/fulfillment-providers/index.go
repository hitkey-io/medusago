// Package fulfillmentproviders mirrors api/admin/fulfillment-providers.
package fulfillmentproviders

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// ListConfig mirrors listTransformQueryConfig.
var ListConfig = httputils.QueryConfig{Defaults: []string{"id", "is_enabled"}, IsList: true, Entity: "fulfillment_provider"}

// AdminFulfillmentProvidersParams mirrors AdminFulfillmentProvidersParams.
var AdminFulfillmentProvidersParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(50), Offset: utils.Num(0)}).Merge(zod.Object(
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("stock_location_id", utils.StringOrStrings().Optional()),
	zod.F("is_enabled", commonvalidators.BooleanString().Optional()),
	zod.F("q", zod.String().Optional()),
)).Strict()

// Middlewares mirrors adminFulfillmentProvidersRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/fulfillment-providers", Middlewares: []medusahttp.MiddlewareFunction{
		httputils.ValidateAndTransformQuery(AdminFulfillmentProvidersParams, ListConfig),
		httputils.MaybeApplyLinkFilter(httputils.LinkFilterOptions{EntryPoint: "location_fulfillment_provider", ResourceID: "fulfillment_provider_id", FilterableField: "stock_location_id"}),
	}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("fulfillment_provider", "fulfillment_providers")}
