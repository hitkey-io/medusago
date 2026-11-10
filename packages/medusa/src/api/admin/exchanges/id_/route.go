// Package id_ mirrors api/admin/exchanges/[id].
package id_

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET /admin/exchanges/:id.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	exchange, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "order_exchange", req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	if exchange == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Exchange with id: %s was not found", req.Params["id"]))
	}
	out := dal.NewRecord()
	out.Set("exchange", exchange)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
