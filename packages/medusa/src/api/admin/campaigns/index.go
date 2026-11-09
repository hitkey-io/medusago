// Package campaigns mirrors api/admin/campaigns.
package campaigns

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultAdminCampaignFields mirrors defaultAdminCampaignFields.
var DefaultAdminCampaignFields = []string{"id", "name", "description", "currency", "campaign_identifier", "*budget", "starts_at", "ends_at", "created_at", "updated_at", "deleted_at"}

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultAdminCampaignFields, Entity: "campaign"}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultAdminCampaignFields, IsList: true, Entity: "campaign"}
)

// AdminGetCampaignsParamsFields mirrors AdminGetCampaignsParamsFields.
var AdminGetCampaignsParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("campaign_identifier", zod.String().Optional()),
	zod.F("budget", zod.Object(zod.F("currency_code", zod.String().Optional())).Optional()),
).Strict()

// AdminGetCampaignsParams mirrors AdminGetCampaignsParams.
var AdminGetCampaignsParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(50)}).
	Merge(AdminGetCampaignsParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetCampaignsParamsFields).Strict())

// Middlewares mirrors adminCampaignRoutesMiddlewares (reads).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/campaigns", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetCampaignsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/campaigns/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("campaign", "campaigns")}
