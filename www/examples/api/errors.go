package api

import (
	"log/slog"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// region: throw
// searchGET returns an INVALID_DATA error when the q query parameter is
// missing. The error handler turns it into a 400 response.
func searchGET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	q, _ := queryString(req.Query, "q")
	if q == "" {
		return common.NewMedusaError(
			common.MedusaErrorTypes.INVALID_DATA,
			"The `q` query parameter is required.",
		)
	}

	return res.JSON(map[string]any{"q": q})
}

// endregion: throw

// region: with-code
func quotaGET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	// The third argument is an optional code, returned as the code property.
	return common.NewMedusaError(
		common.MedusaErrorTypes.NOT_ALLOWED,
		"The quota was reached.",
		"quota_reached",
	)
}

// endregion: with-code

// region: report
// withErrorReporting wraps a route handler: it reports the errors the
// handler returns, then returns them for the error handler to process.
func withErrorReporting(handler medusahttp.RouteHandler) medusahttp.RouteHandler {
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
		err := handler(req, res)
		if err != nil {
			// for example, capture the error in a third-party service
			slog.Error("route failed", "path", req.Path(), "error", err)
		}
		return err
	}
}

// endregion: report

// ErrorRoutes are the route files of the errors chapter.
var ErrorRoutes = []medusahttp.RouteFile{
	{Path: "/custom/search", Route: medusahttp.Route{GET: withErrorReporting(searchGET)}},
	{Path: "/custom/quota", Route: medusahttp.Route{GET: quotaGET}},
}
