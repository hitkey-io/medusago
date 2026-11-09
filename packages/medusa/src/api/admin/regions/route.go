// Package regions mirrors api/admin/regions.
package regions

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/region/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "region", httputils.ListVariables(req.FilterableFields, req.QueryConfig.Pagination), req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return res.JSON(httputils.ListResponse("regions", r))
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	result, err := workflows.CreateRegionsWorkflow.Run(req.Ctx(), req.Scope, workflows.CreateRegionsInput{Regions: []*dal.Record{body}})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	region, err := RefetchRegion(req.Ctx(), req.Scope, id, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return Respond(res, region)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST}
