package api

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
)

// region: path-param
func helloByID(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return res.JSON(map[string]any{
		"message": fmt.Sprintf("[GET] Hello %s!", req.Params["id"]),
	})
}

// endregion: path-param

// region: multiple-path-params
func helloByIDAndName(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return res.JSON(map[string]any{
		"message": fmt.Sprintf("[GET] Hello %s - %s!", req.Params["id"], req.Params["name"]),
	})
}

// endregion: multiple-path-params

// region: query-param
func greetGET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	name, _ := queryString(req.Query, "name")

	return res.JSON(map[string]any{
		"message": fmt.Sprintf("Hello %s", name),
	})
}

// endregion: query-param

// region: body-param
func greetPOST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	value, _ := body(req.Body).Get("name")
	name, _ := value.(string)

	return res.JSON(map[string]any{
		"message": fmt.Sprintf("[POST] Hello %s!", name),
	})
}

// endregion: body-param

// region: routes
// ParameterRoutes are the route files of the parameters chapter.
var ParameterRoutes = []medusahttp.RouteFile{
	{Path: "/hello-world/:id", Route: medusahttp.Route{GET: helloByID}},
	{Path: "/hello-world/:id/name/:name", Route: medusahttp.Route{GET: helloByIDAndName}},
	{Path: "/greeting", Route: medusahttp.Route{GET: greetGET, POST: greetPOST}},
}

// endregion: routes
