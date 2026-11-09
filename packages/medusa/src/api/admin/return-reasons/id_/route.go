// Package id_ mirrors api/admin/return-reasons/[id]/route.ts.
package id_

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	r, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "return_reason", req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	if r == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Return reason with id: %s was not found", req.Params["id"]))
	}
	out := dal.NewRecord()
	out.Set("return_reason", r)
	return res.JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
