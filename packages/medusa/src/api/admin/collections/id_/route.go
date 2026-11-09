// Package id_ mirrors api/admin/collections/[id]/route.ts.
package id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	collections "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/collections"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	entity, err := collections.Refetch(req.Ctx(), req.Scope, req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return collections.Respond(res, "collection", entity)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	existing, err := collections.Refetch(req.Ctx(), req.Scope, req.Params["id"], []string{"id"})
	if err != nil {
		return err
	}
	if existing == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`Collection with id "%s" not found`, req.Params["id"]))
	}
	body, _ := req.ValidatedBody.(*dal.Record)
	selector := dal.NewRecord()
	selector.Set("id", req.Params["id"])
	rest, additional := collections.SplitAdditionalData(body)
	_, err = workflows.UpdateCollectionsWorkflow.Run(req.Ctx(), req.Scope, workflows.UpdateCollectionsInput{Selector: selector, Update: rest, AdditionalData: additional})
	if err != nil {
		return err
	}
	entity, err := collections.Refetch(req.Ctx(), req.Scope, req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return collections.Respond(res, "collection", entity)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeleteCollectionsWorkflow.Run(req.Ctx(), req.Scope, workflows.DeleteCollectionsInput{IDs: []any{id}}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "collection")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
