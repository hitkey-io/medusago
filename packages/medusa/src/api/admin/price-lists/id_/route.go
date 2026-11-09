// Package id_ mirrors api/admin/price-lists/[id]/route.ts.
package id_

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	pricelists "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/price-lists"
)

// FetchPriceList mirrors fetchPriceList.
func FetchPriceList(req *medusahttp.MedusaRequest, id string, fields []string) (any, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	pl, err := httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "price_lists", f, fields)
	if err != nil {
		return nil, err
	}
	if pl == nil {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Price list with id: %s was not found", id))
	}
	return pricelists.TransformPriceList(pl), nil
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	pl, err := FetchPriceList(req, req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("price_list", pl)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
