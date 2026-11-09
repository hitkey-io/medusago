// Package auth mirrors api/cloud/auth/route.ts.
package auth

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET: cloud authentication is enabled when the user actor
// accepts the "cloud" auth method (not configurable in the Go port yet).
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	out := dal.NewRecord()
	out.Set("enabled", false)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
