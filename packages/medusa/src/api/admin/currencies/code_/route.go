// Package code_ mirrors api/admin/currencies/[code]/route.ts.
package code_

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filters := dal.NewRecord()
	filters.Set("code", req.Params["code"])
	vars := dal.NewRecord()
	vars.Set("filters", filters)
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "currency", vars, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	if len(r.Data) == 0 {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Currency with code: %s was not found", req.Params["code"]))
	}
	out := dal.NewRecord()
	out.Set("currency", r.Data[0])
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
