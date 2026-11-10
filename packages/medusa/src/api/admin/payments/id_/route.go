// Package id_ mirrors api/admin/payments/[id].
package id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/payments"
)

// GET mirrors GET /admin/payments/:id.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return payments.Respond(req, res, req.Params["id"])
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
