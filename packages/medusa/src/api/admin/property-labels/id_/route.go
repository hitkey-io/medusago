// Package id_ mirrors api/admin/property-labels/[id].
package id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/settings/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/settings/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	propertylabels "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/property-labels"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	pl, err := propertylabels.ByID(req, id)
	if err != nil {
		return err
	}
	if pl == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Property label with id: %s not found", id))
	}
	out := dal.NewRecord()
	out.Set("property_label", pl)
	return res.JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	body, _ := req.ValidatedBody.(*dal.Record)
	update := dal.NewRecord()
	update.Set("id", id)
	rest := propertylabels.Pick(body, "label", "description")
	for _, k := range rest.Keys() {
		v, _ := rest.Get(k)
		update.Set(k, v)
	}
	if _, err := workflows.UpdatePropertyLabelsWorkflow.Run(req.Ctx(), req.Scope, steps.UpdatePropertyLabelsStepInput{PropertyLabels: []*dal.Record{update}}); err != nil {
		return err
	}
	pl, err := propertylabels.ByID(req, id)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if pl != nil {
		out.Set("property_label", pl)
	}
	return res.JSON(out)
}

// DELETE mirrors DELETE (the label is hard deleted).
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeletePropertyLabelsWorkflow.Run(req.Ctx(), req.Scope, steps.DeletePropertyLabelsStepInput{IDs: []any{id}}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "property_label")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
