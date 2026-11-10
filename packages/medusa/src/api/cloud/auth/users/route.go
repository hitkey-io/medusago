// Package users mirrors api/cloud/auth/users.
package users

import (
	authsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/auth/steps"
	authworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/auth/workflows"
	userworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/user/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

// POST mirrors POST: a Medusa Cloud identity gets (or is linked to) a user.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	ac := req.AuthContext
	if a := get(ac, "actor_id"); a != nil && a != "" {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "The user is already registered and cannot create a new account.")
	}
	meta, _ := get(ac, "user_metadata").(*dal.Record)
	email := get(meta, "email")
	if email == nil || email == "" {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Email is required to create a user account.")
	}
	authIdentityID := get(ac, "auth_identity_id")
	f := dal.NewRecord()
	f.Set("id", authIdentityID)
	identityAny, err := httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "auth_identity", f, []string{"id", "provider_identities.provider"})
	identity := asRecord(identityAny)
	if err != nil {
		return err
	}
	pis, _ := get(identity, "provider_identities").([]any)
	if len(pis) != 1 || get(asRecord(pis[0]), "provider") != "cloud" {
		return common.NewMedusaError(common.MedusaErrorTypes.UNAUTHORIZED, "Only cloud identities can create a user account")
	}
	uf := dal.NewRecord()
	uf.Set("email", email)
	userAny, err := httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "user", uf, []string{"id"})
	user := asRecord(userAny)
	if err != nil {
		return err
	}
	if user != nil {
		if _, err := authworkflows.SetAuthAppMetadataWorkflow.Run(req.Ctx(), req.Scope, authsteps.SetAuthAppMetadataStepInput{AuthIdentityID: authIdentityID, ActorType: "user", Value: get(user, "id")}); err != nil {
			return err
		}
		idf := dal.NewRecord()
		idf.Set("id", get(user, "id"))
		updated, err := httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "user", idf, []string{"*"})
		if err != nil {
			return err
		}
		out := dal.NewRecord()
		out.Set("user", updated)
		return res.Status(200).JSON(out)
	}
	data := dal.NewRecord()
	data.Set("email", email)
	data.Set("first_name", get(meta, "given_name"))
	data.Set("last_name", get(meta, "family_name"))
	created, err := userworkflows.CreateUserAccountWorkflow.Run(req.Ctx(), req.Scope, userworkflows.CreateUserAccountWorkflowInput{AuthIdentityID: authIdentityID, UserData: data})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("user", created)
	return res.Status(200).JSON(out)
}

func asRecord(v any) *dal.Record { r, _ := v.(*dal.Record); return r }

// Route is the route file.
var Route = medusahttp.Route{POST: POST}

// Middlewares mirrors cloudRoutesMiddlewares (users).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/cloud/auth/users", Middlewares: []medusahttp.MiddlewareFunction{
		middlewares.Authenticate([]string{"user"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{AllowUnregistered: true}),
	}},
}
