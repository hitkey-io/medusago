// Package confirm mirrors api/auth/verification/confirm.
package confirm

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/mfa"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	b := mfa.Body(req)
	provider, _ := b.Get("code_provider")
	r, err := mfa.AuthService(req).ConfirmAuthVerification(req.Ctx(), mfa.Str(b, "code"), provider)
	if err != nil {
		return err
	}
	get := func(k string) any { v, _ := r.Get(k); return v }
	out := dal.NewRecord()
	out.Set("entity_id", get("entity_id"))
	out.Set("entity_type", get("entity_type"))
	out.Set("code_provider", get("code_provider"))
	out.Set("verified_at", get("verified_at"))
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
