// Package id_ mirrors api/admin/reservations/[id]/route.ts.
package id_

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/reservation/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/reservations"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	update := dal.NewRecord()
	if body, ok := req.ValidatedBody.(*dal.Record); ok {
		for _, k := range body.Keys() {
			v, _ := body.Get(k)
			update.Set(k, v)
		}
	}
	update.Set("id", id)
	if _, err := workflows.UpdateReservationsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{update}); err != nil {
		return err
	}
	reservation, err := reservations.RefetchReservation(req.Ctx(), id, req.Scope, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("reservation", reservation)
	return res.Status(200).JSON(out)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeleteReservationsWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "reservation")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: httputils.RemoteQueryRetrieveRoute("reservation", "reservation", "Reservation with id: %s was not found"), POST: POST, DELETE: DELETE}
