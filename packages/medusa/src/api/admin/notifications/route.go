package notifications

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	r, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{
		Entity: "notification", IDOrFilter: req.FilterableFields, Fields: req.QueryConfig.Fields, Pagination: req.QueryConfig.Pagination,
	})
	if err != nil {
		return err
	}
	return res.JSON(httputils.ListResponse("notifications", r))
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
