// Package id_ mirrors api/admin/promotions/[id]/route.ts.
package id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/promotion/workflows"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/campaigns"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/promotions"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET: by id or code.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	idOrCode := req.Params["id"]
	byID := dal.NewRecord()
	byID.Set("id", idOrCode)
	byCode := dal.NewRecord()
	byCode.Set("code", idOrCode)
	f := dal.NewRecord()
	f.Set("$or", []any{byID, byCode})
	p, err := httputils.RemoteQueryFirst(req.Ctx(), req.Scope, "promotion", f, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	if p == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Promotion with id or code: %s was not found", idOrCode))
	}
	out := dal.NewRecord()
	out.Set("promotion", p)
	return res.Status(200).JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	body, _ := req.ValidatedBody.(*dal.Record)
	rest, additional := campaigns.SplitAdditionalData(body)
	data := dal.NewRecord()
	data.Set("id", id)
	for _, k := range rest.Keys() {
		v, _ := rest.Get(k)
		data.Set(k, v)
	}
	if _, err := workflows.UpdatePromotionsWorkflow.Run(req.Ctx(), req.Scope, workflows.PromotionsInput{PromotionsData: []*dal.Record{data}, AdditionalData: additional}); err != nil {
		return err
	}
	p, err := promotions.RefetchPromotion(req.Ctx(), id, req.Scope, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("promotion", p)
	return res.Status(200).JSON(out)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeletePromotionsWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "promotion")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
