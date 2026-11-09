package saleschannels

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/sales-channel/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "sales_channels", httputils.ListVariables(req.FilterableFields, req.QueryConfig.Pagination), req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return res.JSON(httputils.ListResponse("sales_channels", r))
}

// Route is the route file.
// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	result, err := workflows.CreateSalesChannelsWorkflow.Run(req.Ctx(), req.Scope, workflows.CreateSalesChannelsInput{SalesChannelsData: []*dal.Record{body}})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	v, err := RefetchSalesChannel(req, id, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return Respond(res, v)
}

var Route = medusahttp.Route{GET: GET, POST: POST}
