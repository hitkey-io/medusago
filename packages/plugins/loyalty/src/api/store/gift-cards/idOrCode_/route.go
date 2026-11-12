// Package idOrCode_ mirrors api/store/gift-cards/[idOrCode] of
// @medusajs/loyalty-plugin.
package idOrCode_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET /store/gift-cards/:idOrCode: the gift card of the code.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	code := req.Params["idOrCode"]
	if code == "" {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_ARGUMENT, "Code is required")
	}
	f := dal.NewRecord()
	f.Set("code", code)
	r, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{Entity: "gift_cards", IDOrFilter: f, Fields: req.QueryConfig.Fields})
	if err != nil {
		return err
	}
	if len(r.Data) == 0 || r.Data[0] == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, "Gift card not found")
	}
	out := dal.NewRecord()
	out.Set("gift_card", r.Data[0])
	return res.JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
