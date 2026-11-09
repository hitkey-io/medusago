package campaigns

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/promotion/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

func isPresent(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case []any:
		return len(t) > 0
	case *dal.Record:
		return len(t.Keys()) > 0
	}
	return !zod.IsUndefined(v)
}

func field(v any, k string) any {
	r, _ := v.(*dal.Record)
	if r == nil {
		return nil
	}
	x, _ := r.Get(k)
	return x
}

func refine(cond func(v any) bool, path, message string) func(v any, add func(*zod.Issue)) {
	return func(v any, add func(*zod.Issue)) {
		if !cond(v) {
			add(&zod.Issue{Path: []any{path}, Message: message})
		}
	}
}

// CreateCampaignBudget mirrors CreateCampaignBudget.
var CreateCampaignBudget = zod.Object(
	zod.F("type", zod.Enum("spend", "usage", "use_by_attribute", "spend_by_attribute")),
	zod.F("limit", zod.Number().Nullish()),
	zod.F("currency_code", zod.String().Nullish()),
	zod.F("attribute", zod.String().Nullish()),
).Strict().
	SuperRefine(refine(func(v any) bool { return field(v, "type") != "spend" || isPresent(field(v, "currency_code")) },
		"currency_code", "currency_code is required when budget type is spend")).
	SuperRefine(refine(func(v any) bool { return field(v, "type") != "usage" || !isPresent(field(v, "currency_code")) },
		"currency_code", "currency_code should not be present when budget type is usage")).
	SuperRefine(refine(func(v any) bool {
		t := field(v, "type")
		return isPresent(field(v, "attribute")) || (t != "use_by_attribute" && t != "spend_by_attribute")
	}, "attribute", "campaign budget attribute is required when budget type is USE_BY_ATTRIBUTE or SPEND_BY_ATTRIBUTE"))

// UpdateCampaignBudget mirrors UpdateCampaignBudget.
var UpdateCampaignBudget = zod.Object(
	zod.F("limit", zod.Number().Nullish()),
).Strict()

// CreateCampaign mirrors CreateCampaign.
var CreateCampaign = zod.Object(
	zod.F("name", zod.String()),
	zod.F("campaign_identifier", zod.String()),
	zod.F("description", zod.String().Nullish()),
	zod.F("budget", CreateCampaignBudget.Nullish()),
	zod.F("starts_at", zod.CoerceDate().Nullish()),
	zod.F("ends_at", zod.CoerceDate().Nullish()),
).Strict()

// AdminCreateCampaign mirrors AdminCreateCampaign.
var AdminCreateCampaign = utils.WithAdditionalData(CreateCampaign)

// UpdateCampaign mirrors UpdateCampaign.
var UpdateCampaign = zod.Object(
	zod.F("name", zod.String().Optional()),
	zod.F("campaign_identifier", zod.String().Optional()),
	zod.F("description", zod.String().Nullish()),
	zod.F("budget", UpdateCampaignBudget.Optional()),
	zod.F("starts_at", zod.CoerceDate().Nullish()),
	zod.F("ends_at", zod.CoerceDate().Nullish()),
)

// AdminUpdateCampaign mirrors AdminUpdateCampaign.
var AdminUpdateCampaign = utils.WithAdditionalData(UpdateCampaign)

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// WriteMiddlewares mirrors the write entries of adminCampaignRoutesMiddlewares
// (authorize is RBAC and not ported).
var WriteMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/campaigns", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateCampaign)),
		httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig),
	)},
	{Methods: []string{"POST"}, Matcher: "/admin/campaigns/:id", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateCampaign)),
		httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig),
	)},
	{Methods: []string{"POST"}, Matcher: "/admin/campaigns/:id/promotions", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(utils.CreateLinkBody())),
		httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig),
	)},
}

func init() {
	Middlewares = append(Middlewares, WriteMiddlewares...)
}

// RefetchCampaign mirrors refetchCampaign.
func RefetchCampaign(ctx context.Context, id any, scope medusahttp.MedusaContainer, fields []string) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	return httputils.RemoteQueryFirst(ctx, scope, "campaign", f, fields)
}

// SplitAdditionalData mirrors const { additional_data, ...rest } = body.
func SplitAdditionalData(body *dal.Record) (*dal.Record, *dal.Record) {
	rest := dal.NewRecord()
	var additional *dal.Record
	if body == nil {
		return rest, nil
	}
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		if k == "additional_data" {
			additional, _ = v.(*dal.Record)
			continue
		}
		rest.Set(k, v)
	}
	return rest, additional
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	rest, additional := SplitAdditionalData(body)
	result, err := workflows.CreateCampaignsWorkflow.Run(req.Ctx(), req.Scope, workflows.CampaignsInput{CampaignsData: []*dal.Record{rest}, AdditionalData: additional})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	campaign, err := RefetchCampaign(req.Ctx(), fmt.Sprint(id), req.Scope, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("campaign", campaign)
	return res.Status(200).JSON(out)
}
