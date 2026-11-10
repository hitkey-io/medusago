// Package id_ mirrors api/admin/claims/[id].
package id_

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET /admin/claims/:id.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	claim, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "order_claim", req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	if claim == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Claim with id: %s was not found", req.Params["id"]))
	}
	out := dal.NewRecord()
	out.Set("claim", claim)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
