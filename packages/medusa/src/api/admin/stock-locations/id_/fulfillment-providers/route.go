// Package fulfillmentproviders mirrors api/admin/stock-locations/[id]/fulfillment-providers/route.ts.
package fulfillmentproviders

import (
	commonworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	stocklocations "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stock-locations"
)

func buildLinks(id string, providers []string) []*dal.Record {
	out := make([]*dal.Record, len(providers))
	for i, p := range providers {
		out[i] = modulessdk.NewLinkDefinition("stock_location", "stock_location_id", id, "fulfillment", "fulfillment_provider_id", p)
	}
	return out
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	add, _ := body.Get("add")
	remove, _ := body.Get("remove")
	id := req.Params["id"]
	if _, err := commonworkflows.BatchLinksWorkflow.Run(req.Ctx(), req.Scope, commonworkflows.BatchLinksInput{Create: buildLinks(id, stocklocations.Strings(add)), Delete: buildLinks(id, stocklocations.Strings(remove))}); err != nil {
		return err
	}
	return stocklocations.Respond(req, res, id)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
