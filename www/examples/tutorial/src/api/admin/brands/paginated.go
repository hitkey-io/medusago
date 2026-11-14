package brands

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	apiutils "github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// region: handler
// ListPaginated lists the brands with pagination, for the admin dashboard.
func ListPaginated(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	result, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{
		Entity: "brand",
		// the fields and pagination that passed the query configuration
		Fields:     req.QueryConfig.Fields,
		Pagination: req.QueryConfig.Pagination,
	}, joiner.Options{})
	if err != nil {
		return err
	}

	// { brands, count, offset, limit }
	return res.JSON(httputils.ListResponse("brands", result))
}

// endregion: handler

// region: middlewares
// ListQueryConfig sets the default fields of GET /admin/brands.
var ListQueryConfig = httputils.QueryConfig{
	Defaults: []string{"id", "name", "products.*"},
	IsList:   true,
}

// ListMiddlewares validate the query parameters of GET /admin/brands: fields,
// limit, offset and order.
var ListMiddlewares = []medusahttp.MiddlewareRoute{
	{
		Matcher: "/admin/brands",
		Methods: []string{"GET"},
		Middlewares: []medusahttp.MiddlewareFunction{
			httputils.ValidateAndTransformQuery(
				apiutils.CreateFindParams(apiutils.FindParamsOptions{}),
				ListQueryConfig,
			),
		},
	},
}

// endregion: middlewares
