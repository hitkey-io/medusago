// Package id_ mirrors api/admin/notifications/[id]/route.ts.
package id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET (a missing notification is answered with {}).
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	n, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "notification", req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if n != nil {
		out.Set("notification", n)
	}
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
