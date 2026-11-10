// Package id_ mirrors api/auth/mfa/factors/[id].
package id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/mfa"
)

// DELETE mirrors DELETE: disables the factor.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	svc := mfa.AuthService(req)
	id := req.Params["id"]
	factor, err := svc.RetrieveAuthMfa(req.Ctx(), id, mfa.AuthIdentityID(req))
	if err != nil {
		return err
	}
	disabled, err := svc.DisableAuthMfa(req.Ctx(), id)
	if err != nil {
		return err
	}
	before, _ := factor.Get("status")
	after, _ := disabled.Get("status")
	if before != "disabled" && after == "disabled" {
		data := dal.NewRecord()
		aid, _ := disabled.Get("auth_identity_id")
		did, _ := disabled.Get("id")
		prov, _ := disabled.Get("provider")
		data.Set("auth_identity_id", aid)
		data.Set("mfa_id", did)
		data.Set("provider", prov)
		if err := mfa.Emit(req.Ctx(), req, "auth.mfa_disabled", data); err != nil {
			return err
		}
	}
	out := dal.NewRecord()
	out.Set("mfa_factor", disabled)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{DELETE: DELETE}
