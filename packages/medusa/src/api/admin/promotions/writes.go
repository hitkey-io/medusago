package promotions

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/promotion/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/promotion/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/campaigns"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// DefaultAdminPromotionRuleFields mirrors defaultAdminPromotionRuleFields.
var DefaultAdminPromotionRuleFields = []string{"id", "description", "attribute", "operator", "values.value"}

// RetrieveRuleConfig mirrors retrieveRuleTransformQueryConfig.
var RetrieveRuleConfig = httputils.QueryConfig{Defaults: DefaultAdminPromotionRuleFields}

var ruleOperator = zod.Enum("gte", "lte", "gt", "lt", "eq", "ne", "in")

// AdminCreatePromotionRule mirrors AdminCreatePromotionRule.
var AdminCreatePromotionRule = zod.Object(
	zod.F("operator", ruleOperator),
	zod.F("description", zod.String().Nullish()),
	zod.F("attribute", zod.String()),
	zod.F("values", utils.StringOrStrings()),
).Strict()

// AdminUpdatePromotionRule mirrors AdminUpdatePromotionRule.
var AdminUpdatePromotionRule = zod.Object(
	zod.F("id", zod.String()),
	zod.F("operator", ruleOperator.Optional()),
	zod.F("description", zod.String().Nullish()),
	zod.F("attribute", zod.String().Optional()),
	zod.F("values", utils.StringOrStrings()),
).Strict()

var (
	methodType = zod.Enum("fixed", "percentage")
	targetType = zod.Enum("order", "shipping_methods", "items")
	allocation = zod.Enum("each", "across", "once")
	// targetSelectionStrategy mirrors z.nativeEnum(ApplicationMethodTargetSelectionStrategy).
	targetSelectionStrategy = zod.Enum("most_expensive_first", "cheapest_first")
	promoType               = zod.Enum("standard", "buyget")
	promoStatus             = zod.Enum("draft", "active", "inactive")
	limitSchema             = func() *zod.Schema { return zod.Number().Int().Min(1).Nullable().Optional() }
	metadataType            = func() *zod.Schema { return zod.Record(zod.String(), zod.Unknown()).Nullish() }
)

// AdminCreateApplicationMethod mirrors AdminCreateApplicationMethod.
var AdminCreateApplicationMethod = zod.Object(
	zod.F("description", zod.String().Nullish()),
	zod.F("value", zod.Number()),
	zod.F("currency_code", zod.String().Nullish()),
	zod.F("max_quantity", zod.Number().Nullish()),
	zod.F("type", methodType),
	zod.F("target_type", targetType),
	zod.F("allocation", allocation.Optional()),
	zod.F("target_rules", zod.Array(AdminCreatePromotionRule).Optional()),
	zod.F("buy_rules", zod.Array(AdminCreatePromotionRule).Optional()),
	zod.F("apply_to_quantity", zod.Number().Nullish()),
	zod.F("buy_rules_min_quantity", zod.Number().Nullish()),
	zod.F("target_selection_strategy", targetSelectionStrategy.Optional()),
).Strict()

// AdminUpdateApplicationMethod mirrors AdminUpdateApplicationMethod.
var AdminUpdateApplicationMethod = zod.Object(
	zod.F("description", zod.String().Nullish()),
	zod.F("value", zod.Number().Optional()),
	zod.F("max_quantity", zod.Number().Nullish()),
	zod.F("currency_code", zod.String().Nullish()),
	zod.F("type", methodType.Optional()),
	zod.F("target_type", targetType.Optional()),
	zod.F("allocation", allocation.Optional()),
	zod.F("apply_to_quantity", zod.Number().Nullish()),
	zod.F("buy_rules_min_quantity", zod.Number().Nullish()),
	zod.F("target_selection_strategy", targetSelectionStrategy.Optional()),
).Strict()

func field(v any, k string) any {
	r, _ := v.(*dal.Record)
	if r == nil {
		return zod.Undefined
	}
	x, ok := r.Get(k)
	if !ok {
		return zod.Undefined
	}
	return x
}

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case bool:
		return t
	case float64:
		return t != 0
	}
	return !zod.IsUndefined(v)
}

// promoRefinement mirrors promoRefinement.
func promoRefinement(v any) bool {
	if truthy(field(v, "campaign")) && truthy(field(v, "campaign_id")) {
		return false
	}
	if field(v, "type") == "buyget" {
		am := field(v, "application_method")
		buy, _ := field(am, "buy_rules").([]any)
		return len(buy) > 0 && !zod.IsUndefined(field(am, "apply_to_quantity")) && !zod.IsUndefined(field(am, "buy_rules_min_quantity"))
	}
	return true
}

func automaticLimit(v any, add func(*zod.Issue)) {
	limit := field(v, "limit")
	if truthy(field(v, "is_automatic")) && limit != nil && !zod.IsUndefined(limit) {
		add(&zod.Issue{Path: []any{"limit"}, Message: "Automatic promotions cannot have a usage limit"})
	}
}

// AdminCreatePromotion mirrors AdminCreatePromotion.
var AdminCreatePromotion = utils.WithAdditionalData(zod.Object(
	zod.F("code", zod.String()),
	zod.F("is_automatic", zod.Boolean().Optional()),
	zod.F("type", promoType),
	zod.F("is_tax_inclusive", zod.Boolean().Optional()),
	zod.F("status", promoStatus.Default("draft")),
	zod.F("campaign_id", zod.String().Nullish()),
	zod.F("campaign", campaigns.CreateCampaign.Optional()),
	zod.F("application_method", AdminCreateApplicationMethod),
	zod.F("rules", zod.Array(AdminCreatePromotionRule).Optional()),
	zod.F("limit", limitSchema()),
	zod.F("metadata", metadataType()),
).Strict()).
	Refine(promoRefinement, "Buyget promotions require at least one buy rule and quantities to be defined").
	SuperRefine(automaticLimit)

// AdminUpdatePromotion mirrors AdminUpdatePromotion.
var AdminUpdatePromotion = utils.WithAdditionalData(zod.Object(
	zod.F("code", zod.String().Optional()),
	zod.F("is_automatic", zod.Boolean().Optional()),
	zod.F("is_tax_inclusive", zod.Boolean().Optional()),
	zod.F("type", promoType.Optional()),
	zod.F("status", promoStatus.Optional()),
	zod.F("campaign_id", zod.String().Nullish()),
	zod.F("application_method", AdminUpdateApplicationMethod.Optional()),
	zod.F("limit", limitSchema()),
	zod.F("metadata", metadataType()),
).Strict()).SuperRefine(automaticLimit)

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

func batchBody() medusahttp.MiddlewareFunction {
	return httputils.ValidateAndTransformBody(httputils.Schema(utils.CreateBatchBody(AdminCreatePromotionRule, AdminUpdatePromotionRule, nil)))
}

// AdminGetPromotionRuleParams mirrors AdminGetPromotionRuleParams.
var AdminGetPromotionRuleParams = zod.Object(
	zod.F("promotion_type", zod.String().Optional()),
	zod.F("application_method_type", zod.String().Optional()),
	zod.F("application_method_target_type", zod.String().Optional()),
)

// AdminGetPromotionRuleTypeParams mirrors AdminGetPromotionRuleTypeParams.
var AdminGetPromotionRuleTypeParams = utils.CreateSelectParams().Merge(AdminGetPromotionRuleParams)

// AdminGetPromotionsRuleValueParams mirrors AdminGetPromotionsRuleValueParams.
var AdminGetPromotionsRuleValueParams = utils.CreateFindParams(utils.FindParamsOptions{Limit: utils.Num(100), Offset: utils.Num(0)}).Merge(zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("value", utils.StringOrStrings().Optional()),
	zod.F("application_method_target_type", zod.String().Optional()),
))

// Rule query configs.
var (
	ListRuleConfig      = httputils.QueryConfig{Defaults: DefaultAdminPromotionRuleFields, IsList: true}
	ListRuleValueConfig = httputils.QueryConfig{Defaults: []string{}, Allowed: []string{}, IsList: true}
)

// WriteMiddlewares mirrors the write entries of
// adminPromotionRoutesMiddlewares (authorize is RBAC and not ported).
var WriteMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/promotions", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(AdminCreatePromotion)),
		httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig),
	)},
	{Methods: []string{"POST"}, Matcher: "/admin/promotions/:id", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdatePromotion)),
		httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig),
	)},
	{Methods: []string{"GET"}, Matcher: "/admin/promotions/:id/:rule_type", Middlewares: mw(httputils.ValidateAndTransformQuery(AdminGetPromotionRuleTypeParams, RetrieveConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/promotions/:id/rules/batch", Middlewares: mw(batchBody(), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveRuleConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/promotions/:id/target-rules/batch", Middlewares: mw(batchBody(), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveRuleConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/promotions/:id/buy-rules/batch", Middlewares: mw(batchBody(), httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveRuleConfig))},
	{Methods: []string{"GET"}, Matcher: "/admin/promotions/rule-value-options/:rule_type/:rule_attribute_id", Middlewares: mw(httputils.ValidateAndTransformQuery(AdminGetPromotionsRuleValueParams, ListRuleValueConfig))},
	{Methods: []string{"GET"}, Matcher: "/admin/promotions/rule-attribute-options/:rule_type", Middlewares: mw(httputils.ValidateAndTransformQuery(AdminGetPromotionRuleParams, ListRuleConfig))},
}

func init() {
	Middlewares = append(Middlewares, WriteMiddlewares...)
}

// RefetchPromotion mirrors refetchPromotion.
func RefetchPromotion(ctx context.Context, id any, scope medusahttp.MedusaContainer, fields []string) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	return httputils.RemoteQueryFirst(ctx, scope, "promotion", f, fields)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	rest, additional := campaigns.SplitAdditionalData(body)
	result, err := workflows.CreatePromotionsWorkflow.Run(req.Ctx(), req.Scope, workflows.PromotionsInput{PromotionsData: []*dal.Record{rest}, AdditionalData: additional})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	promotion, err := RefetchPromotion(req.Ctx(), fmt.Sprint(id), req.Scope, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("promotion", promotion)
	return res.Status(200).JSON(out)
}

func records(v any) []*dal.Record {
	list, _ := v.([]any)
	out := []*dal.Record{}
	for _, x := range list {
		if r, ok := x.(*dal.Record); ok {
			out = append(out, r)
		}
	}
	return out
}

// BatchRulesRoute builds the POST handler of the rule batch routes.
func BatchRulesRoute(ruleType steps.RuleType) medusahttp.RouteHandler {
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
		in := workflows.BatchPromotionRulesInput{ID: req.Params["id"], RuleType: ruleType}
		if body, ok := req.ValidatedBody.(*dal.Record); ok {
			if v, ok := body.Get("create"); ok {
				in.Create = records(v)
			}
			if v, ok := body.Get("update"); ok {
				in.Update = records(v)
			}
			if v, ok := body.Get("delete"); ok {
				in.Delete, _ = v.([]any)
			}
		}
		result, err := workflows.BatchPromotionRulesWorkflow.Run(req.Ctx(), req.Scope, in)
		if err != nil {
			return err
		}
		fetch := func(rows []*dal.Record) ([]any, error) {
			if len(rows) == 0 {
				return []any{}, nil
			}
			ids := make([]any, len(rows))
			for i, r := range rows {
				ids[i], _ = r.Get("id")
			}
			f := dal.NewRecord()
			f.Set("id", ids)
			vars := dal.NewRecord()
			vars.Set("filters", f)
			r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "promotion_rule", vars, req.QueryConfig.Fields)
			if err != nil {
				return nil, err
			}
			if r.Data == nil {
				return []any{}, nil
			}
			return r.Data, nil
		}
		created, err := fetch(result.Created)
		if err != nil {
			return err
		}
		updated, err := fetch(result.Updated)
		if err != nil {
			return err
		}
		deletedIDs := result.Deleted
		if deletedIDs == nil {
			deletedIDs = []any{}
		}
		deleted := dal.NewRecord()
		deleted.Set("ids", deletedIDs)
		deleted.Set("object", "promotion-rule")
		deleted.Set("deleted", true)
		out := dal.NewRecord()
		out.Set("created", created)
		out.Set("updated", updated)
		out.Set("deleted", deleted)
		return res.Status(200).JSON(out)
	}
}
