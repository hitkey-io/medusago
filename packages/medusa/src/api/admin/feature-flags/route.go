// Package featureflags mirrors api/admin/feature-flags/route.ts.
package featureflags

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	ff "github.com/hitkey-io/medusago/packages/core/utils/src/feature-flags"
)

var authenticate = false

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	router, _ := req.Scope.Resolve(common.ContainerRegistrationKeys.FEATURE_FLAG_ROUTER).(*ff.FlagRouter)
	flags := dal.NewRecord()
	if router != nil {
		flags = router.ListFlags()
	}
	out := dal.NewRecord()
	out.Set("feature_flags", flags)
	return res.JSON(out)
}

// Route is the route file (AUTHENTICATE = false).
var Route = medusahttp.Route{GET: GET, Authenticate: &authenticate}
