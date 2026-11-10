// Package factors mirrors api/auth/mfa/factors.
package factors

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/mfa"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	f := dal.NewRecord()
	f.Set("auth_identity_id", mfa.AuthIdentityID(req))
	list, err := mfa.AuthService(req).ListAuthMfa(req.Ctx(), f)
	if err != nil {
		return err
	}
	items := make([]any, len(list))
	for i, x := range list {
		items[i] = x
	}
	out := dal.NewRecord()
	out.Set("mfa_factors", items)
	return res.Status(200).JSON(out)
}

// POST mirrors POST: starts a factor enrollment.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	b := mfa.Body(req)
	label, _ := b.Get("label")
	meta, _ := b.Get("metadata")
	metadata, _ := meta.(*dal.Record)
	r, err := mfa.AuthService(req).StartAuthMfa(req.Ctx(), mfa.AuthIdentityID(req), mfa.Str(b, "provider"), label, mfa.Str(b, "issuer"), metadata)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("mfa_factor", r.Mfa)
	out.Set("secret", r.Secret)
	out.Set("otpauth_url", r.OtpauthURL)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST}
