// Package id_ mirrors api/admin/refund-reasons/[id]/route.ts.
package id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	r, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "refund_reason", req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if r != nil {
		out.Set("refund_reason", r)
	}
	return res.JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
