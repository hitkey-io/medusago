// Package id_ mirrors api/admin/sales-channels/[id]/route.ts.
package id_

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// RefetchSalesChannel mirrors refetchSalesChannel.
func RefetchSalesChannel(req *medusahttp.MedusaRequest, id string, fields []string) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	vars := dal.NewRecord()
	vars.Set("filters", f)
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "sales_channel", vars, fields)
	if err != nil || len(r.Data) == 0 {
		return nil, err
	}
	return r.Data[0], nil
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	sc, err := RefetchSalesChannel(req, req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	if sc == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Sales channel with id: %s not found", req.Params["id"]))
	}
	out := dal.NewRecord()
	out.Set("sales_channel", sc)
	return res.JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
