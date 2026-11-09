// Package promotions mirrors api/admin/promotions.
package promotions

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// DefaultAdminPromotionFields mirrors defaultAdminPromotionFields.
var DefaultAdminPromotionFields = []string{"id", "code", "is_automatic", "is_tax_inclusive", "type", "limit", "used", "status", "created_at", "updated_at", "deleted_at", "metadata", "*campaign", "*campaign.budget", "*application_method", "*application_method.buy_rules", "application_method.buy_rules.values.value", "*application_method.target_rules", "application_method.target_rules.values.value", "rules.id", "rules.attribute", "rules.operator", "rules.values.value"}

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: DefaultAdminPromotionFields, Entity: "promotion"}
	ListConfig     = httputils.QueryConfig{Defaults: DefaultAdminPromotionFields, IsList: true, Entity: "promotion"}
)

func stringsOrOperators() *zod.Schema {
	return zod.Union(zod.String(), zod.Array(zod.String()), utils.CreateOperatorMap(nil, nil))
}

// AdminGetPromotionsParamsFields mirrors AdminGetPromotionsParamsFields.
var AdminGetPromotionsParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("code", stringsOrOperators().Optional()),
	zod.F("id", stringsOrOperators().Optional()),
	zod.F("campaign_id", utils.StringOrStrings().Optional()),
	zod.F("application_method", zod.Object(zod.F("currency_code", utils.StringOrStrings().Optional())).Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
)

// AdminGetPromotionsParams mirrors AdminGetPromotionsParams.
var AdminGetPromotionsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(50), Offset: utils.Num(0)}).
	Merge(AdminGetPromotionsParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetPromotionsParamsFields)).Strict()

// Middlewares mirrors adminPromotionRoutesMiddlewares (reads).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/promotions", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetPromotionsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/promotions/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("promotion", "promotions")}
