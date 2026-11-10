// Package update mirrors api/auth/[actor_type]/[auth_provider]/update.
package update

import (
	"context"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	authtypes "github.com/hitkey-io/medusago/packages/core/types/src/auth"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

type providerUpdater interface {
	UpdateProvider(ctx context.Context, provider string, data *dal.Record) authtypes.AuthenticationResponse
}

// POST mirrors POST: updates the provider identity of the token's entity.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	svc, _ := req.Scope.Resolve(utilsmodulessdk.AUTH).(providerUpdater)
	data := dal.NewRecord()
	if body, ok := req.Body.(*dal.Record); ok {
		for _, k := range body.Keys() {
			v, _ := body.Get(k)
			data.Set(k, v)
		}
	}
	var actorID any
	if req.AuthContext != nil {
		actorID, _ = req.AuthContext.Get("actor_id")
	}
	data.Set("entity_id", actorID)
	r := svc.UpdateProvider(req.Ctx(), req.Params["auth_provider"], data)
	if r.Success && r.AuthIdentity != nil {
		out := dal.NewRecord()
		out.Set("success", true)
		return res.Status(200).JSON(out)
	}
	msg := r.Error
	if msg == "" {
		msg = "Unauthorized"
	}
	return common.NewMedusaError(common.MedusaErrorTypes.UNAUTHORIZED, msg)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
