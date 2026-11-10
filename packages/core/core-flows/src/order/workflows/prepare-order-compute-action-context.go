package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// PrepareOrderComputeActionContextInput mirrors PrepareOrderComputeActionContextStepInput.
type PrepareOrderComputeActionContextInput struct {
	Order          *dal.Record
	PreviewedOrder *dal.Record
}

func normalizeProductContext(product *dal.Record) *dal.Record {
	if product == nil {
		return nil
	}
	out := dal.NewRecord()
	out.Set("id", g(product, "id"))
	for _, k := range []string{"collection_id", "tags", "categories", "type_id"} {
		if v := g(product, k); v != nil {
			out.Set(k, v)
		}
	}
	return out
}

func filterComputeAdjustments(adjustments any) []any {
	out := []any{}
	for _, a := range recs(adjustments) {
		if truthy(g(a, "id")) && truthy(g(a, "code")) {
			r := dal.NewRecord()
			r.Set("id", g(a, "id"))
			r.Set("code", g(a, "code"))
			out = append(out, r)
		}
	}
	return out
}

func firstPresent(a, b any) any {
	if a != nil {
		return a
	}
	return b
}

// PrepareOrderComputeActionContextStep mirrors prepareOrderComputeActionContextStep.
var PrepareOrderComputeActionContextStep = composer.CreateStep("prepare-order-compute-action-context",
	func(ctx composer.StepExecutionContext, in PrepareOrderComputeActionContextInput) (composer.StepResponse[*dal.Record], error) {
		order, preview := in.Order, in.PreviewedOrder
		items := recs(firstPresent(g(preview, "items"), g(order, "items")))
		methods := recs(firstPresent(g(preview, "shipping_methods"), g(order, "shipping_methods")))

		var productIDs, optionIDs []any
		seenP, seenO := map[any]bool{}, map[any]bool{}
		for _, it := range items {
			if id := g(it, "product_id"); truthy(id) && !seenP[id] {
				seenP[id] = true
				productIDs = append(productIDs, id)
			}
		}
		for _, m := range methods {
			if id := g(m, "shipping_option_id"); truthy(id) && !seenO[id] {
				seenO[id] = true
				optionIDs = append(optionIDs, id)
			}
		}
		q := commonsteps.QueryOf(ctx.Container)
		products := map[any]*dal.Record{}
		if len(productIDs) > 0 {
			f := dal.NewRecord()
			f.Set("id", productIDs)
			res, err := q.Graph(ctx.Context, query.GraphInput{Entity: "product", Fields: []string{"id", "collection_id", "tags.id", "categories.id", "type_id"}, Filters: f}, joiner.Options{})
			if err != nil {
				return composer.StepResponse[*dal.Record]{}, err
			}
			for _, p := range recs(res.Data) {
				products[g(p, "id")] = p
			}
		}
		options := map[any]*dal.Record{}
		if len(optionIDs) > 0 {
			f := dal.NewRecord()
			f.Set("id", optionIDs)
			res, err := q.Graph(ctx.Context, query.GraphInput{Entity: "shipping_option", Fields: []string{"id", "shipping_option_type_id"}, Filters: f}, joiner.Options{})
			if err != nil {
				return composer.StepResponse[*dal.Record]{}, err
			}
			for _, o := range recs(res.Data) {
				options[g(o, "id")] = o
			}
		}

		computedItems := make([]any, 0, len(items))
		for _, it := range items {
			c := copyRec(it)
			product := rec(g(it, "product"))
			if product == nil && truthy(g(it, "product_id")) {
				product = products[g(it, "product_id")]
			}
			np := normalizeProductContext(product)
			if adj := filterComputeAdjustments(g(it, "adjustments")); len(adj) > 0 {
				c.Set("adjustments", adj)
			} else {
				c.Delete("adjustments")
			}
			if np == nil && truthy(g(it, "product_id")) {
				np = dal.NewRecord()
				np.Set("id", g(it, "product_id"))
			}
			if np != nil {
				c.Set("product", np)
			} else {
				c.Delete("product")
			}
			computedItems = append(computedItems, c)
		}
		computedMethods := make([]any, 0, len(methods))
		for _, m := range methods {
			c := copyRec(m)
			var option *dal.Record
			if truthy(g(m, "shipping_option_id")) {
				option = options[g(m, "shipping_option_id")]
			}
			if adj := filterComputeAdjustments(g(m, "adjustments")); len(adj) > 0 {
				c.Set("adjustments", adj)
			} else {
				c.Delete("adjustments")
			}
			typeID := firstPresent(g(rec(g(m, "shipping_option")), "shipping_option_type_id"), g(option, "shipping_option_type_id"))
			if truthy(typeID) {
				so := dal.NewRecord()
				so.Set("shipping_option_type_id", typeID)
				c.Set("shipping_option", so)
			} else {
				c.Delete("shipping_option")
			}
			computedMethods = append(computedMethods, c)
		}

		out := dal.NewRecord()
		out.Set("currency_code", firstPresent(g(preview, "currency_code"), g(order, "currency_code")))
		pc, oc := rec(g(preview, "customer")), rec(g(order, "customer"))
		if truthy(g(pc, "id")) || truthy(g(oc, "id")) || truthy(g(order, "customer_id")) {
			cust := dal.NewRecord()
			cust.Set("id", firstPresent(g(pc, "id"), firstPresent(g(oc, "id"), g(order, "customer_id"))))
			if groups := firstPresent(g(pc, "groups"), g(oc, "groups")); groups != nil {
				list := []any{}
				for _, gr := range recs(groups) {
					r := dal.NewRecord()
					r.Set("id", g(gr, "id"))
					list = append(list, r)
				}
				cust.Set("groups", list)
			}
			out.Set("customer", cust)
		}
		pr, or := rec(g(preview, "region")), rec(g(order, "region"))
		if truthy(g(pr, "id")) || truthy(g(or, "id")) || truthy(g(order, "region_id")) {
			r := dal.NewRecord()
			r.Set("id", firstPresent(g(pr, "id"), firstPresent(g(or, "id"), g(order, "region_id"))))
			out.Set("region", r)
		}
		addr := rec(firstPresent(g(preview, "shipping_address"), g(order, "shipping_address")))
		if truthy(g(addr, "country_code")) {
			a := dal.NewRecord()
			a.Set("country_code", g(addr, "country_code"))
			out.Set("shipping_address", a)
		}
		if v := firstPresent(g(preview, "sales_channel_id"), g(order, "sales_channel_id")); v != nil {
			out.Set("sales_channel_id", v)
		}
		if v := firstPresent(g(preview, "email"), g(order, "email")); v != nil {
			out.Set("email", v)
		}
		out.Set("items", computedItems)
		out.Set("shipping_methods", computedMethods)
		return composer.NewStepResponse(out), nil
	}, nil)
