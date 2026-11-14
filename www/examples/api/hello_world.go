package api

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
)

// region: handlers
// helloGET handles GET /hello-world.
func helloGET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return res.JSON(map[string]any{
		"message": "[GET] Hello world!",
	})
}

// helloPOST handles POST /hello-world.
func helloPOST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return res.JSON(map[string]any{
		"message": "[POST] Hello world!",
	})
}

// endregion: handlers

// region: route-file
// HelloWorldRoute adds two API routes: GET /hello-world and POST /hello-world.
var HelloWorldRoute = medusahttp.RouteFile{
	Path: "/hello-world",
	Route: medusahttp.Route{
		GET:  helloGET,
		POST: helloPOST,
	},
}

// endregion: route-file
