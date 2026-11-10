// Package details mirrors api/admin/index/details.
package details

import (
	"context"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

type infoGetter interface {
	GetInfo(ctx context.Context) ([]any, error)
}

// GET mirrors GET: the index information of every indexed entity.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	index, _ := req.Scope.Resolve(utilsmodulessdk.INDEX).(infoGetter)
	info, err := index.GetInfo(req.Ctx())
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("metadata", info)
	return res.JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
