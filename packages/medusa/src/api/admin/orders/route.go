package orders

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filters := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			filters.Set(k, v)
		}
	}
	filters.Set("is_draft_order", false)
	r, err := workflows.GetOrdersList(req.Ctx(), httputils.QueryOf(req.Scope), workflows.GetOrdersListInput{
		Fields:    req.QueryConfig.Fields,
		Variables: httputils.ListVariables(filters, req.QueryConfig.Pagination),
	})
	if err != nil {
		return err
	}
	if r.Metadata == nil {
		r.Metadata = dal.NewRecord()
	}
	return res.JSON(httputils.ListResponse("orders", r))
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
