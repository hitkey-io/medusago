// Package id_ mirrors api/admin/stores/[id]/route.ts.
package id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/store/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stores"
)

// GET mirrors GET (the id is a top-level variable, as upstream).
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	vars := dal.NewRecord()
	vars.Set("id", req.Params["id"])
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "store", vars, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	var v any
	if len(r.Data) > 0 {
		v = r.Data[0]
	}
	return stores.Respond(res, v)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	existing, err := stores.RefetchStore(req, id, []string{"id"})
	if err != nil {
		return err
	}
	if existing == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`Store with id "%s" not found`, id))
	}
	body, _ := req.ValidatedBody.(*dal.Record)
	selector := dal.NewRecord()
	selector.Set("id", id)
	result, err := workflows.UpdateStoresWorkflow.Run(req.Ctx(), req.Scope, workflows.UpdateStoresInput{Selector: selector, Update: body})
	if err != nil {
		return err
	}
	var rid any
	if len(result) > 0 {
		rid, _ = result[0].Get("id")
	}
	v, err := stores.RefetchStore(req, rid, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return stores.Respond(res, v)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST}
