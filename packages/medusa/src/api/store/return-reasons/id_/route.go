// Package id_ mirrors api/store/return-reasons/[id]/route.ts.
package id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET (a missing reason is answered with {}).
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	vars := dal.NewRecord()
	vars.Set("id", req.Params["id"])
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "return_reason", vars, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if len(r.Data) > 0 && r.Data[0] != nil {
		out.Set("return_reason", r.Data[0])
	}
	return res.JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
