// Package providers mirrors api/auth/[actor_type]/providers/route.ts.
package providers

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	authutils "github.com/hitkey-io/medusago/packages/medusa/src/api/auth/utils"
	"github.com/hitkey-io/medusago/packages/modules/auth/src/services"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	service := req.Scope.Resolve(utilsmodulessdk.AUTH).(*services.AuthModuleService)
	allowed := authutils.GetAllowedAuthProvidersForActor(authutils.ConfigOf(req.Scope), req.Params["actor_type"])
	providers := service.ListAuthProviders(allowed)
	out := dal.NewRecord()
	out.Set("providers", providers)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
