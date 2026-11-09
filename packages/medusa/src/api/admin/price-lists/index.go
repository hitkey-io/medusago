// Package pricelists mirrors api/admin/price-lists.
package pricelists

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
	commonvalidators "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/common-validators"
)

// Field lists.
var (
	AdminPriceListPriceQueryFields  = []string{"id", "currency_code", "amount", "min_quantity", "max_quantity", "created_at", "deleted_at", "updated_at", "price_set.variant.id", "price_rules.value", "price_rules.attribute"}
	AdminPriceListRemoteQueryFields = []string{"id", "type", "description", "title", "status", "starts_at", "ends_at", "created_at", "updated_at", "deleted_at", "metadata", "price_list_rules.value", "price_list_rules.attribute"}
)

// Query configs.
var (
	RetrievePriceConfig = httputils.QueryConfig{Defaults: AdminPriceListPriceQueryFields, Entity: "price_list"}
	ListPriceConfig     = httputils.QueryConfig{Defaults: AdminPriceListPriceQueryFields, IsList: true, Entity: "price_list"}
	RetrieveConfig      = httputils.QueryConfig{Defaults: AdminPriceListRemoteQueryFields, Entity: "price_list"}
	ListConfig          = httputils.QueryConfig{Defaults: AdminPriceListRemoteQueryFields, IsList: true, Entity: "price_list"}
)

// AdminGetPriceListsParamsFields mirrors AdminGetPriceListsParamsFields.
var AdminGetPriceListsParamsFields = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("id", utils.StringOrStrings().Optional()),
	zod.F("starts_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("ends_at", utils.CreateOperatorMap(nil, nil).Optional()),
	zod.F("status", zod.Array(zod.Enum("active", "draft")).Optional()),
	zod.F("rules_count", zod.Array(zod.Number()).Optional()),
)

// AdminGetPriceListsParams mirrors AdminGetPriceListsParams.
var AdminGetPriceListsParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(50)}).
	Merge(AdminGetPriceListsParamsFields).Merge(commonvalidators.ApplyAndAndOrOperators(AdminGetPriceListsParamsFields))

// AdminGetPriceListPricesParams mirrors AdminGetPriceListPricesParams.
var AdminGetPriceListPricesParams = utils.CreateFindParams(utils.FindParamsOptions{Offset: utils.Num(0), Limit: utils.Num(50)})

// Middlewares mirrors adminPriceListsRoutesMiddlewares (reads).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/price-lists", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetPriceListsParams, ListConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/price-lists/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(utils.CreateSelectParams(), RetrieveConfig)}},
}

func recordList(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case []*dal.Record:
		out := make([]any, len(t))
		for i, r := range t {
			out[i] = r
		}
		return out
	}
	return nil
}

// TransformPriceList mirrors transformPriceList (buildPriceListRules and
// buildPriceSetPricesForCore).
func TransformPriceList(v any) any {
	pl, ok := v.(*dal.Record)
	if !ok || pl == nil {
		return v
	}
	if rules, ok := pl.Get("price_list_rules"); ok && rules != nil {
		out := dal.NewRecord()
		for _, x := range recordList(rules) {
			r, _ := x.(*dal.Record)
			if r == nil {
				continue
			}
			attr, _ := r.Get("attribute")
			val, _ := r.Get("value")
			if val == nil {
				val = []any{}
			}
			out.Set(attrString(attr), val)
		}
		pl.Set("rules", out)
	} else {
		pl.Delete("rules")
	}
	if prices, ok := pl.Get("prices"); ok && prices != nil {
		var out []any
		for _, x := range recordList(prices) {
			p, _ := x.(*dal.Record)
			if p == nil {
				continue
			}
			n := dal.NewRecord()
			var variantID any
			if ps, ok := p.Get("price_set"); ok {
				if psr, _ := ps.(*dal.Record); psr != nil {
					if vr, ok := psr.Get("variant"); ok {
						if vrec, _ := vr.(*dal.Record); vrec != nil {
							variantID, _ = vrec.Get("id")
						}
					}
				}
			}
			pr, hasRules := p.Get("price_rules")
			for _, k := range p.Keys() {
				if k == "price_rules" || k == "price_set" {
					continue
				}
				val, _ := p.Get(k)
				n.Set(k, val)
			}
			if variantID != nil {
				n.Set("variant_id", variantID)
			}
			if hasRules {
				rules := dal.NewRecord()
				for _, y := range recordList(pr) {
					r, _ := y.(*dal.Record)
					if r == nil {
						continue
					}
					attr, _ := r.Get("attribute")
					val, _ := r.Get("value")
					rules.Set(attrString(attr), val)
				}
				n.Set("rules", rules)
			}
			out = append(out, n)
		}
		if out == nil {
			out = []any{}
		}
		pl.Set("prices", out)
	} else {
		pl.Delete("prices")
	}
	pl.Delete("price_list_rules")
	return pl
}

func attrString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return "undefined"
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "price_list", httputils.ListVariables(req.FilterableFields, req.QueryConfig.Pagination), req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	for i, pl := range r.Data {
		r.Data[i] = TransformPriceList(pl)
	}
	return res.JSON(httputils.ListResponse("price_lists", r))
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
