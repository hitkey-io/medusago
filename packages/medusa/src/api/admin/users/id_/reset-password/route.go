// Package resetpassword mirrors api/admin/users/[id]/reset-password.
package resetpassword

import (
	"fmt"

	authworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/auth/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	authutils "github.com/hitkey-io/medusago/packages/medusa/src/api/auth/utils"
)

// POST mirrors POST: a reset token for the user's emailpass identity.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	f := dal.NewRecord()
	f.Set("id", id)
	user, err := httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "user", f, []string{"email"})
	if err != nil {
		return err
	}
	u, _ := user.(*dal.Record)
	if u == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("User with id: %s was not found", id))
	}
	email, _ := u.Get("email")
	token, err := authworkflows.GenerateResetPasswordTokenWorkflow.Run(req.Ctx(), req.Scope, authworkflows.GenerateResetPasswordTokenInput{
		EntityID: fmt.Sprint(email), ActorType: "user", Provider: "emailpass", Secret: authutils.ConfigOf(req.Scope).ProjectConfig.HTTP.JwtSecret,
	})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("token", token)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
