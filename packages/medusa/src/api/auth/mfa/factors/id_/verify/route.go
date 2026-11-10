// Package verify mirrors api/auth/mfa/factors/[id]/verify.
package verify

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/mfa"
)

// POST mirrors POST: verifies the factor setup.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	svc := mfa.AuthService(req)
	id := req.Params["id"]
	factor, err := svc.RetrieveAuthMfa(req.Ctx(), id, mfa.AuthIdentityID(req))
	if err != nil {
		return err
	}
	verified, err := svc.VerifyAuthMfa(req.Ctx(), id, mfa.AuthIdentityID(req), mfa.Str(mfa.Body(req), "code"))
	if err != nil {
		return err
	}
	before, _ := factor.Get("status")
	after, _ := verified.Get("status")
	if before != "enabled" && after == "enabled" {
		data := dal.NewRecord()
		aid, _ := verified.Get("auth_identity_id")
		vid, _ := verified.Get("id")
		prov, _ := verified.Get("provider")
		data.Set("auth_identity_id", aid)
		data.Set("mfa_id", vid)
		data.Set("provider", prov)
		if err := mfa.Emit(req.Ctx(), req, "auth.mfa_enabled", data); err != nil {
			return err
		}
	}
	out := dal.NewRecord()
	out.Set("mfa_factor", verified)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
