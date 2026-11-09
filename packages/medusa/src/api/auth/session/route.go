// Package session mirrors api/auth/session/route.ts.
package session

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	authutils "github.com/hitkey-io/medusago/packages/medusa/src/api/auth/utils"
)

// POST mirrors POST: the auth context is stored in the session.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	req.Session.Set("auth_context", req.AuthContext)
	out := dal.NewRecord()
	out.Set("user", req.AuthContext)
	return res.Status(200).JSON(out)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	cfg := authutils.ConfigOf(req.Scope).ProjectConfig
	name := "connect.sid"
	if cfg.SessionOptions != nil && cfg.SessionOptions.Name != "" {
		name = cfg.SessionOptions.Name
	}
	err := req.Session.Destroy()
	res.ClearCookie(name, medusahttp.CookieOptionsFromConfig(cfg.CookieOptions))
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("success", true)
	return res.JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST, DELETE: DELETE}
