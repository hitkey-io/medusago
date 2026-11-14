package api

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	apiutils "github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// region: query-config
// ProductListConfig restricts what GET /store/custom can retrieve.
var ProductListConfig = httputils.QueryConfig{
	// The fields retrieved when the request doesn't send a fields parameter.
	Defaults: []string{"id", "title"},
	// The route returns a list, with limit and offset pagination.
	IsList: true,
	// The only fields and relations a client can request.
	Allowed: []string{"id", "title", "handle", "variants", "variants.title", "variants.sku"},
}

// endregion: query-config

// region: middlewares
// StoreCustomMiddlewares validate the query parameters of GET /store/custom.
var StoreCustomMiddlewares = []medusahttp.MiddlewareRoute{
	{
		Matcher: "/store/custom",
		Methods: []string{"GET"},
		Middlewares: []medusahttp.MiddlewareFunction{
			httputils.ValidateAndTransformQuery(
				apiutils.CreateFindParams(apiutils.FindParamsOptions{}),
				ProductListConfig,
			),
		},
	},
}

// endregion: middlewares

// region: handler
func storeCustomGETProducts(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filters := dal.NewRecord()
	filters.Set("status", "published")

	result, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{
		Entity:     "product",
		IDOrFilter: filters,
		// The fields and pagination that passed the query config.
		Fields:     req.QueryConfig.Fields,
		Pagination: req.QueryConfig.Pagination,
	})
	if err != nil {
		return err
	}

	// { products, count, offset, limit }
	return res.JSON(httputils.ListResponse("products", result))
}

// endregion: handler

// StoreCustomRoute is the route file of /store/custom.
var StoreCustomRoute = medusahttp.RouteFile{
	Path:  "/store/custom",
	Route: medusahttp.Route{GET: storeCustomGETProducts},
}

// region: relations-limit
// ProductListConfigWithLimit allows requests to expand one relation only,
// whatever http.storeRelationsLimit is.
var ProductListConfigWithLimit = httputils.QueryConfig{
	Defaults:            []string{"id", "title"},
	IsList:              true,
	Allowed:             []string{"id", "title", "variants", "variants.title", "variants.options"},
	StoreRelationsLimit: 1,
}

// endregion: relations-limit

// region: disallowed
// ProductListConfigWithDisallowed removes any requested field that has one
// of these segments, even when it's in the allowed list.
var ProductListConfigWithDisallowed = httputils.QueryConfig{
	Defaults:   []string{"id", "title"},
	IsList:     true,
	Allowed:    []string{"id", "title", "variants", "variants.prices"},
	Disallowed: []string{"prices"},
}

// endregion: disallowed

// region: allow-fields
// AllowFieldsMiddleware allows retrieving the b2b_company relation, a custom
// linked data model, with Medusa's GET /store/customers/me route.
var AllowFieldsMiddleware = medusahttp.MiddlewareRoute{
	Matcher: "/store/customers/me",
	Middlewares: []medusahttp.MiddlewareFunction{
		middlewares.AllowFields([]string{"b2b_company"}),
	},
}

// endregion: allow-fields
