// Package id_ mirrors api/admin/products/[id]/route.ts.
package id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	p, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "product", req.Params["id"], products.RemapKeysForProduct(req.QueryConfig.Fields))
	if err != nil {
		return err
	}
	if p == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, "Product not found")
	}
	out := dal.NewRecord()
	out.Set("product", products.RemapProductResponse(p))
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
