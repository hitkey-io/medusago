// Package saleschannels mirrors api/admin/stock-locations/[id]/sales-channels/route.ts.
package saleschannels

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/stock-location/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	stocklocations "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stock-locations"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	add, _ := body.Get("add")
	remove, _ := body.Get("remove")
	if _, err := workflows.LinkSalesChannelsToStockLocationWorkflow.Run(req.Ctx(), req.Scope, workflows.LinkSalesChannelsInput{ID: req.Params["id"], Add: stocklocations.Strings(add), Remove: stocklocations.Strings(remove)}); err != nil {
		return err
	}
	return stocklocations.Respond(req, res, req.Params["id"])
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
