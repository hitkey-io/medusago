// Package products mirrors api/admin/price-lists/[id]/products/route.ts.
package products

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/price-list/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	pricelists "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-lists"
)

// fetchPriceListPriceIdsForProduct mirrors fetchPriceListPriceIdsForProduct.
func fetchPriceListPriceIdsForProduct(ctx context.Context, scope medusahttp.MedusaContainer, priceListID string, productIDs []any) ([]any, error) {
	f := dal.NewRecord()
	f.Set("product_id", productIDs)
	vars := dal.NewRecord()
	vars.Set("filters", f)
	variants, err := httputils.RemoteQueryList(ctx, scope, "variants", vars, []string{"price_set.id"})
	if err != nil {
		return nil, err
	}
	priceSetIDs := []any{}
	for _, v := range variants.Data {
		r, _ := v.(*dal.Record)
		if r == nil {
			continue
		}
		ps, _ := r.Get("price_set")
		if psr, _ := ps.(*dal.Record); psr != nil {
			if id, _ := psr.Get("id"); id != nil && id != "" {
				priceSetIDs = append(priceSetIDs, id)
			}
		}
	}
	pf := dal.NewRecord()
	pf.Set("price_set_id", priceSetIDs)
	pf.Set("price_list_id", priceListID)
	pvars := dal.NewRecord()
	pvars.Set("filters", pf)
	prices, err := httputils.RemoteQueryList(ctx, scope, "prices", pvars, []string{"id"})
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, p := range prices.Data {
		if r, _ := p.(*dal.Record); r != nil {
			id, _ := r.Get("id")
			out = append(out, id)
		}
	}
	return out, nil
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	var remove []any
	if body, ok := req.ValidatedBody.(*dal.Record); ok {
		v, _ := body.Get("remove")
		remove, _ = v.([]any)
	}
	if len(remove) == 0 {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "No product ids passed to remove from price list")
	}
	priceIDs, err := fetchPriceListPriceIdsForProduct(req.Ctx(), req.Scope, id, remove)
	if err != nil {
		return err
	}
	if _, err := workflows.BatchPriceListPricesWorkflow.Run(req.Ctx(), req.Scope, workflows.BatchPriceListPricesInput{ID: id, Create: []*dal.Record{}, Update: []*dal.Record{}, Delete: priceIDs}); err != nil {
		return err
	}
	pl, err := pricelists.FetchPriceList(req.Ctx(), req.Scope, id, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("price_list", pl)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
