package saleschannels

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// AdminSalesChannelRoutesMiddlewares mirrors
// adminSalesChannelRoutesMiddlewares (ported routes only).
var AdminSalesChannelRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/sales-channels", Middlewares: []medusahttp.MiddlewareFunction{
		httputils.ValidateAndTransformQuery(AdminGetSalesChannelsParams, ListTransformQueryConfig),
		httputils.MaybeApplyLinkFilter(httputils.LinkFilterOptions{EntryPoint: "sales_channel_location", ResourceID: "sales_channel_id", FilterableField: "location_id"}),
		httputils.MaybeApplyLinkFilter(httputils.LinkFilterOptions{EntryPoint: "publishable_api_key_sales_channel", ResourceID: "sales_channel_id", FilterableField: "publishable_key_id"}),
	}},
	{Methods: []string{"GET"}, Matcher: "/admin/sales-channels/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetSalesChannelParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/sales-channels", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateSalesChannel)), httputils.ValidateAndTransformQuery(AdminGetSalesChannelParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/sales-channels/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateSalesChannel)), httputils.ValidateAndTransformQuery(AdminGetSalesChannelParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/sales-channels/:id/products", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(utils.CreateLinkBody())), httputils.ValidateAndTransformQuery(AdminGetSalesChannelParams, RetrieveTransformQueryConfig)}},
}
