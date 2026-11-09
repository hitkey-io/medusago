// Package shippingoptions mirrors api/admin/shipping-options.
package shippingoptions

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// Defaults mirrors defaultAdminShippingOptionFields.
var Defaults = []string{"id", "name", "price_type", "data", "provider_id", "metadata", "created_at", "updated_at", "*rules", "*type", "*prices", "*prices.price_rules", "*service_zone", "*shipping_profile", "*provider"}

// Query configs.
var (
	RetrieveConfig = httputils.QueryConfig{Defaults: Defaults, Entity: "shipping_option"}
	ListConfig     = httputils.QueryConfig{Defaults: Defaults, IsList: true, Entity: "shipping_option"}
)

// AdminGetShippingOptionsParams mirrors AdminGetShippingOptionsParams.
var AdminGetShippingOptionsParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(20)}).Merge(zod.Object(
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("q", zod.String().Optional()),
	zod.F("service_zone_id", utils.StringOrStrings().Optional()),
	zod.F("shipping_profile_id", utils.StringOrStrings().Optional()),
	zod.F("provider_id", utils.StringOrStrings().Optional()),
	zod.F("stock_location_id", utils.StringOrStrings().Optional()),
	zod.F("is_return", commonvalidators.BooleanString().Optional()),
	zod.F("admin_only", commonvalidators.BooleanString().Optional()),
	zod.F("shipping_option_type_id", utils.StringOrStrings().Optional()),
	zod.F("created_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("updated_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("deleted_at", utils.CreateOperatorMap(nil, nil).Optional()),
))

var ruleOperators = []string{"in", "eq", "ne", "gt", "gte", "lt", "lte", "nin"}

// AdminCreateShippingOptionRule mirrors AdminCreateShippingOptionRule.
var AdminCreateShippingOptionRule = zod.Object(
	zod.F("operator", zod.Enum(ruleOperators...)),
	zod.F("attribute", zod.String()),
	zod.F("value", zod.Union(zod.String(), zod.Array(zod.String()))),
).Strict()

// AdminUpdateShippingOptionRule mirrors AdminUpdateShippingOptionRule.
var AdminUpdateShippingOptionRule = zod.Object(
	zod.F("id", zod.String()),
	zod.F("operator", zod.Enum(ruleOperators...)),
	zod.F("attribute", zod.String()),
	zod.F("value", zod.Union(zod.String(), zod.Array(zod.String()))),
).Strict()

// AdminCreateShippingOptionTypeObject mirrors AdminCreateShippingOptionTypeObject.
var AdminCreateShippingOptionTypeObject = zod.Object(
	zod.F("label", zod.String()),
	zod.F("description", zod.String().Optional()),
	zod.F("code", zod.String()),
).Strict()

var adminPriceRules = zod.Array(zod.Object(
	zod.F("attribute", zod.Literal("item_total")),
	zod.F("operator", zod.Enum("gte", "lte", "gt", "lt", "eq")),
	zod.F("value", zod.Number()),
))

func priceWith(required bool, key string) *zod.Schema {
	keySchema := zod.String()
	amount := zod.Number()
	var fields []zod.Field
	if !required {
		fields = append(fields, zod.F("id", zod.String().Optional()))
		keySchema = keySchema.Optional()
		amount = amount.Optional()
	}
	fields = append(fields, zod.F(key, keySchema), zod.F("amount", amount), zod.F("rules", adminPriceRules.Optional()))
	return zod.Object(fields...).Strict()
}

func defined(r *dal.Record, k string) bool {
	v, ok := r.Get(k)
	return ok && v != nil
}

// AdminCreateShippingOption mirrors AdminCreateShippingOption.
var AdminCreateShippingOption = zod.Object(
	zod.F("name", zod.String()),
	zod.F("service_zone_id", zod.String()),
	zod.F("shipping_profile_id", zod.String()),
	zod.F("data", zod.Record(zod.String(), zod.Unknown()).Optional()),
	zod.F("price_type", zod.Enum("calculated", "flat")),
	zod.F("provider_id", zod.String()),
	zod.F("type", AdminCreateShippingOptionTypeObject.Optional()),
	zod.F("type_id", zod.String().Optional()),
	zod.F("prices", zod.Array(zod.Union(priceWith(true, "currency_code"), priceWith(true, "region_id")))),
	zod.F("rules", zod.Array(AdminCreateShippingOptionRule).Optional()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Optional()),
).Strict().SuperRefine(func(v any, add func(*zod.Issue)) {
	r, ok := v.(*dal.Record)
	if !ok {
		return
	}
	if defined(r, "type_id") == defined(r, "type") {
		add(&zod.Issue{Path: []any{"type_id", "type"}, Message: "Exactly one of 'type' or 'type_id' must be provided, but not both"})
	}
})

// AdminUpdateShippingOption mirrors AdminUpdateShippingOption.
var AdminUpdateShippingOption = zod.Object(
	zod.F("name", zod.String().Optional()),
	zod.F("data", zod.Record(zod.String(), zod.Unknown()).Optional()),
	zod.F("price_type", zod.Enum("calculated", "flat").Optional()),
	zod.F("provider_id", zod.String().Optional()),
	zod.F("shipping_profile_id", zod.String().Optional()),
	zod.F("type", AdminCreateShippingOptionTypeObject.Optional()),
	zod.F("type_id", zod.String().Optional()),
	zod.F("prices", zod.Array(zod.Union(priceWith(false, "currency_code"), priceWith(false, "region_id"))).Optional()),
	zod.F("rules", zod.Array(zod.Union(AdminUpdateShippingOptionRule, AdminCreateShippingOptionRule)).Optional()),
	zod.F("metadata", zod.Record(zod.String(), zod.Unknown()).Optional()),
).Strict().SuperRefine(func(v any, add func(*zod.Issue)) {
	r, ok := v.(*dal.Record)
	if !ok {
		return
	}
	hasType, hasTypeID := defined(r, "type"), defined(r, "type_id")
	if (hasType || hasTypeID) && hasType == hasTypeID {
		add(&zod.Issue{Path: []any{"type_id", "type"}, Message: "Only one of 'type' or 'type_id' can be provided"})
	}
})

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// Middlewares mirrors adminShippingOptionRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/shipping-options", Middlewares: mw(
		httputils.ValidateAndTransformQuery(AdminGetShippingOptionsParams, ListConfig),
		httputils.MaybeApplyLinkFilter(httputils.LinkFilterOptions{EntryPoint: "location_fulfillment_set", ResourceID: "fulfillment_set_id", FilterableField: "stock_location_id", FilterByField: "service_zone.fulfillment_set_id"}),
	)},
	{Methods: []string{"GET"}, Matcher: "/admin/shipping-options/:id", Middlewares: mw(httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/shipping-options", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateShippingOption)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/shipping-options/:id", Middlewares: mw(httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateShippingOption)), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/shipping-options/:id/rules/batch", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(utils.CreateBatchBody(AdminCreateShippingOptionRule, AdminUpdateShippingOptionRule, nil))),
		httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveRuleConfig),
	)},
}

// RetrieveRuleConfig mirrors retrieveRuleTransformQueryConfig.
var RetrieveRuleConfig = httputils.QueryConfig{Defaults: []string{"id", "description", "attribute", "operator", "values.value"}}

// RefetchShippingOption mirrors refetchShippingOption (query.graph).
func RefetchShippingOption(req *medusahttp.MedusaRequest, id any) (any, error) {
	r, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{Entity: "shipping_option", IDOrFilter: filterByID(id), Fields: req.QueryConfig.Fields})
	if err != nil || len(r.Data) == 0 {
		return nil, err
	}
	return r.Data[0], nil
}

func filterByID(id any) *dal.Record {
	f := dal.NewRecord()
	f.Set("id", id)
	return f
}

// Respond writes { shipping_option } with status 200.
func Respond(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id any) error {
	v, err := RefetchShippingOption(req, id)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if v != nil {
		out.Set("shipping_option", v)
	}
	return res.Status(200).JSON(out)
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	withDeleted := req.QueryConfig.WithDeleted != nil && *req.QueryConfig.WithDeleted
	r, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{Entity: "shipping_option", IDOrFilter: req.FilterableFields, Fields: req.QueryConfig.Fields, Pagination: req.QueryConfig.Pagination, WithDeleted: withDeleted})
	if err != nil {
		return err
	}
	return res.JSON(httputils.ListResponse("shipping_options", r))
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	result, err := workflows.CreateShippingOptionsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{body})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	return Respond(req, res, id)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET, POST: POST}
