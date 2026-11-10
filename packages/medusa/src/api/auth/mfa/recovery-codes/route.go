// Package recoverycodes mirrors api/auth/mfa/recovery-codes.
package recoverycodes

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/mfa"
)

// POST mirrors POST: regenerates the recovery codes.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	svc := mfa.AuthService(req)
	f := dal.NewRecord()
	f.Set("auth_identity_id", mfa.AuthIdentityID(req))
	f.Set("status", "enabled")
	factors, err := svc.ListAuthMfa(req.Ctx(), f)
	if err != nil {
		return err
	}
	if len(factors) == 0 {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Recovery codes require an enabled MFA factor")
	}
	var count *int
	if v, ok := mfa.Body(req).Get("count"); ok && v != nil {
		if n, ok := v.(float64); ok {
			c := int(n)
			count = &c
		}
	}
	codes, err := svc.GenerateAuthMfaRecoveryCodes(req.Ctx(), mfa.AuthIdentityID(req), count)
	if err != nil {
		return err
	}
	data := dal.NewRecord()
	data.Set("auth_identity_id", mfa.AuthIdentityID(req))
	data.Set("count", float64(len(codes)))
	if err := mfa.Emit(req.Ctx(), req, "auth.mfa_recovery_codes_generated", data); err != nil {
		return err
	}
	list := make([]any, len(codes))
	for i, c := range codes {
		list[i] = c
	}
	out := dal.NewRecord()
	out.Set("recovery_codes", list)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
