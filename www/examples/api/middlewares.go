package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"regexp"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// region: logger
// logRequests logs every request it runs for, then calls the next
// middleware of the stack.
func logRequests(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
	slog.Info("Received a request!", "method", req.Method, "path", req.Path())

	next(nil)
	return nil
}

// endregion: logger

// region: global-and-route
// MiddlewaresExample shows the two types of middlewares.
var MiddlewaresExample = []medusahttp.MiddlewareRoute{
	// A global middleware: it applies to every method.
	{
		Matcher:     "/custom*",
		Middlewares: []medusahttp.MiddlewareFunction{logRequests},
	},
	// A route middleware: it only applies to POST and PUT requests.
	{
		Matcher:     "/custom*",
		Methods:     []string{"POST", "PUT"},
		Middlewares: []medusahttp.MiddlewareFunction{logRequests},
	},
}

// endregion: global-and-route

// region: regexp
// RegexpMiddleware applies to /custom and every path starting with /custom/.
var RegexpMiddleware = medusahttp.MiddlewareRoute{
	MatcherRe:   regexp.MustCompile(`^/custom(/.*)?$`),
	Middlewares: []medusahttp.MiddlewareFunction{logRequests},
}

// endregion: regexp

// region: path-params
// PathParamMiddleware applies to the routes of /custom/:id.
var PathParamMiddleware = medusahttp.MiddlewareRoute{
	Matcher:     "/custom/:id",
	Middlewares: []medusahttp.MiddlewareFunction{logRequests},
}

// endregion: path-params

// region: reject
// requireApiVersion stops requests that don't send the header
// x-api-version: the route handler doesn't run.
func requireApiVersion(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
	if req.Header.Get("x-api-version") == "" {
		// Returning an error (or passing it to next) skips the remaining
		// middlewares and the route handler, and reaches the error handler.
		return common.NewMedusaError(
			common.MedusaErrorTypes.INVALID_DATA,
			"The x-api-version header is required.",
		)
	}

	next(nil)
	return nil
}

// endregion: reject

// region: body-parser
// verifySignature computes the HMAC of the exact bytes that were received.
func verifySignature(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write(req.RawBody)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(req.Header.Get("x-signature"))) {
		return common.NewMedusaError(common.MedusaErrorTypes.UNAUTHORIZED, "Invalid signature.")
	}

	next(nil)
	return nil
}

// WebhookMiddlewares keep the raw request body and raise the size limit of
// POST /webhooks/custom.
var WebhookMiddlewares = []medusahttp.MiddlewareRoute{
	{
		Matcher: "/webhooks/custom",
		Methods: []string{"POST"},
		BodyParser: &medusahttp.BodyParserConfig{
			SizeLimit:       "500kb",
			PreserveRawBody: true,
		},
		Middlewares: []medusahttp.MiddlewareFunction{verifySignature},
	},
}

// endregion: body-parser

// region: webhook-handler
func webhookPOST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return res.JSON(map[string]any{"received": len(req.RawBody)})
}

// endregion: webhook-handler

// WebhookRoute is the route file of the webhook.
var WebhookRoute = medusahttp.RouteFile{
	Path:  "/webhooks/custom",
	Route: medusahttp.Route{POST: webhookPOST},
}

// region: disable-parser
// RawStreamMiddlewares turn the body parser off, so the handler reads
// req.Request.Body itself.
var RawStreamMiddlewares = []medusahttp.MiddlewareRoute{
	{
		Matcher:    "/webhooks/stream",
		Methods:    []string{"POST"},
		BodyParser: &medusahttp.BodyParserConfig{Disabled: true},
	},
}

// endregion: disable-parser
