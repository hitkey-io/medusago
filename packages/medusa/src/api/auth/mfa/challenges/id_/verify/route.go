// Package verify mirrors api/auth/mfa/challenges/[id]/verify.
package verify

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/mfa"
	authutils "github.com/hitkey-io/medusago/packages/medusa/src/api/auth/utils"
)

// POST mirrors POST: completes the challenge and issues a token.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	svc := mfa.AuthService(req)
	b := mfa.Body(req)
	challenge, err := svc.VerifyAuthMfaChallenge(req.Ctx(), req.Params["id"], mfa.Str(b, "method"), mfa.Str(b, "code"), mfa.AuthIdentityID(req))
	if err != nil {
		return err
	}
	aid, _ := challenge.Get("auth_identity_id")
	if aid == nil || aid == "" {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "MFA challenge is missing an auth identity")
	}
	identity, err := svc.Internal("AuthIdentity").Retrieve(req.Ctx(), aid, utilsmodulessdk.FindConfig{Relations: []string{"provider_identities"}})
	if err != nil {
		return err
	}
	challengeProvider, _ := challenge.Get("auth_provider")
	if p, ok := challengeProvider.(string); ok && p != "" {
		found := false
		pis, _ := identity.Get("provider_identities")
		for _, x := range recordList(pis) {
			if v, _ := x.Get("provider"); v == p {
				found = true
			}
		}
		if !found {
			return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`Auth identity does not have a provider identity for "%s"`, p))
		}
	}
	actorType := req.AuthString("actor_type")
	authProvider := req.AuthString("auth_provider")
	if authProvider == "" {
		authProvider, _ = challengeProvider.(string)
	}
	enabled := true
	completed, _ := challenge.Get("completed_at")
	token, err := authutils.GenerateJwtTokenForAuthIdentity(req.Ctx(), req.Scope, authutils.TokenInput{AuthIdentity: identity, ActorType: actorType, AuthProvider: authProvider, MfaEnabled: &enabled, MfaChallengeCompletedAt: completed})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("token", token)
	return res.Status(200).JSON(out)
}

func recordList(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		var out []*dal.Record
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
