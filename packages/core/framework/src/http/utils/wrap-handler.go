// Package utils mirrors @medusajs/framework/src/http/utils.
package utils

import (
	"net/http"

	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
)

// RouteHandler is a route handler that can fail. Returned errors are rendered
// by the core error handler, as next(err) does upstream.
type RouteHandler func(w http.ResponseWriter, r *http.Request) error

// WrapHandler mirrors wrapHandler: it turns a RouteHandler into an
// http.Handler that forwards errors to the core error handler.
func WrapHandler(fn RouteHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			middlewares.HandleError(w, r, err)
		}
	})
}
