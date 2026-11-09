// Package id_ mirrors api/store/product-variants/[id].
package id_

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	productvariants "github.com/hitkey-io/medusago/packages/medusa/src/api/store/product-variants"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/store/products"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filters := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			filters.Set(k, v)
		}
	}
	filters.Set("id", req.Params["id"])
	r, withInventory, err := productvariants.Graph(req, filters, nil)
	if err != nil {
		return err
	}
	if len(r.Data) == 0 || r.Data[0] == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Product variant with id: %s was not found", req.Params["id"]))
	}
	variant := r.Data[0]
	if withInventory {
		if err := products.WrapInventoryQuantity(req, []any{variant}); err != nil {
			return err
		}
	}
	if err := productvariants.WrapVariantsWithTaxPrices(req, []any{variant}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("variant", variant)
	return res.JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
