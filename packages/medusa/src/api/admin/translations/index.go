// Package translations mirrors api/admin/translations (route.ts,
// middlewares.ts, validators.ts and query-config.ts).
package translations

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
	featureflags "github.com/hitkey-io/medusago/packages/medusa/src/feature-flags"
)

// DefaultAdminTranslationFields mirrors defaultAdminTranslationFields.
var DefaultAdminTranslationFields = []string{"id", "reference_id", "reference", "locale_code", "translations"}

// Query configs.
var (
	RetrieveTransformQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminTranslationFields, Entity: "translation"}
	ListTransformQueryConfig     = httputils.QueryConfig{Defaults: DefaultAdminTranslationFields, Entity: "translation", IsList: true}
)

// AdminGetTranslationParamsFields mirrors AdminGetTranslationParamsFields.
var AdminGetTranslationParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("reference_id", utils.StringOrStrings().Optional()),
	zod.F("reference", zod.String().Optional()),
	zod.F("locale_code", zod.String().Optional()),
)

// AdminGetTranslationsParams mirrors AdminGetTranslationsParams.
var AdminGetTranslationsParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(20), Offset: utils.Num(0)}).
	Merge(AdminGetTranslationParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetTranslationParamsFields))

// AdminCreateTranslation mirrors AdminCreateTranslation.
var AdminCreateTranslation = zod.Object(
	zod.F("reference_id", zod.String()),
	zod.F("reference", zod.String()),
	zod.F("locale_code", zod.String()),
	zod.F("translations", zod.Record(zod.String(), zod.String())),
)

// AdminUpdateTranslation mirrors AdminUpdateTranslation.
var AdminUpdateTranslation = zod.Object(
	zod.F("id", zod.String()),
	zod.F("reference_id", zod.String().Optional()),
	zod.F("reference", zod.String().Optional()),
	zod.F("locale_code", zod.String().Optional()),
	zod.F("translations", zod.Record(zod.String(), zod.String()).Optional()),
)

// AdminBatchTranslations mirrors AdminBatchTranslations.
var AdminBatchTranslations = utils.CreateBatchBody(AdminCreateTranslation, AdminUpdateTranslation, nil)

// AdminTranslationStatisticsParams mirrors AdminTranslationStatisticsParams:
// locales and entity_types normalized to arrays.
var AdminTranslationStatisticsParams = zod.Object(
	zod.F("locales", utils.StringOrStrings()),
	zod.F("entity_types", utils.StringOrStrings()),
).Transform(func(v any) (any, error) {
	r, ok := v.(*dal.Record)
	if !ok {
		return v, nil
	}
	out := dal.NewRecord()
	for _, k := range []string{"locales", "entity_types"} {
		x, _ := r.Get(k)
		if list, isList := x.([]any); isList {
			out.Set(k, list)
		} else {
			out.Set(k, []any{x})
		}
	}
	return out, nil
})

// AdminTranslationSettingsParams mirrors AdminTranslationSettingsParams.
var AdminTranslationSettingsParams = zod.Object(
	zod.F("entity_type", zod.String().Optional()),
	zod.F("is_active", commonvalidators.BooleanString().Optional()),
)

var adminUpdateTranslationSettings = zod.Object(
	zod.F("id", zod.String()),
	zod.F("entity_type", zod.String().Optional()),
	zod.F("fields", zod.Array(zod.String()).Optional()),
	zod.F("is_active", zod.Boolean().Optional()),
)

var adminCreateTranslationSettings = zod.Object(
	zod.F("entity_type", zod.String()),
	zod.F("fields", zod.Array(zod.String())),
	zod.F("is_active", zod.Boolean().Optional()),
)

// AdminBatchTranslationSettings mirrors AdminBatchTranslationSettings.
var AdminBatchTranslationSettings = utils.CreateBatchBody(adminCreateTranslationSettings, adminUpdateTranslationSettings, nil)

// AdminTranslationEntitiesParams mirrors AdminTranslationEntitiesParams.
var AdminTranslationEntitiesParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(20), Offset: utils.Num(0)}).Merge(zod.Object(
	zod.F("type", zod.String()),
	zod.F("id", utils.StringOrStrings().Optional()),
))

// Middlewares mirrors adminTranslationsRoutesMiddlewares (the authorize
// guards are RBAC, not ported).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/translations", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetTranslationsParams, ListTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/translations/batch", BodyParser: &medusahttp.BodyParserConfig{SizeLimit: "2mb" /* DEFAULT_BATCH_ENDPOINTS_SIZE_LIMIT */},
		Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminBatchTranslations))}},
	{Methods: []string{"GET"}, Matcher: "/admin/translations/statistics", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminTranslationStatisticsParams, httputils.QueryConfig{})}},
	{Methods: []string{"GET"}, Matcher: "/admin/translations/settings", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminTranslationSettingsParams, httputils.QueryConfig{})}},
	{Methods: []string{"POST"}, Matcher: "/admin/translations/settings/batch", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminBatchTranslationSettings))}},
	{Methods: []string{"GET"}, Matcher: "/admin/translations/entities", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminTranslationEntitiesParams, ListTransformQueryConfig)}},
}

// FeatureFlag is the flag of the route files (defineFileConfig isDisabled).
var FeatureFlag = featureflags.Translation.Key

// Route is route.ts: GET lists the translations through query.graph.
var Route = medusahttp.Route{GET: httputils.RefetchListRouteWithOptions("translation", "translations", joiner.Options{Cache: caching.Enabled()})}
