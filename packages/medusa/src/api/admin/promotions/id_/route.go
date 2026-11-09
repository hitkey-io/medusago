// Package id_ mirrors api/admin/promotions/[id]/route.ts.
package id_

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET: by id or code.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	idOrCode := req.Params["id"]
	byID := dal.NewRecord()
	byID.Set("id", idOrCode)
	byCode := dal.NewRecord()
	byCode.Set("code", idOrCode)
	f := dal.NewRecord()
	f.Set("$or", []any{byID, byCode})
	p, err := httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "promotion", f, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	if p == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Promotion with id or code: %s was not found", idOrCode))
	}
	out := dal.NewRecord()
	out.Set("promotion", p)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
