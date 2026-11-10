// Package preview mirrors api/admin/orders/[id]/preview.
package preview

import (
	"context"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

type previewer interface {
	PreviewOrderChange(ctx context.Context, orderID string) (*dal.Record, error)
}

// GET mirrors GET /admin/orders/:id/preview.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	svc := req.Scope.Resolve(utilsmodulessdk.ORDER).(previewer)
	order, err := svc.PreviewOrderChange(req.Ctx(), req.Params["id"])
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("order", order)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
