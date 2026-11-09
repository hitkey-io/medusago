package collections

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// POST mirrors POST of route.ts.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	rest, additional := SplitAdditionalData(body)
	result, err := workflows.CreateCollectionsWorkflow.Run(req.Ctx(), req.Scope, workflows.CreateCollectionsInput{Collections: []*dal.Record{rest}, AdditionalData: additional})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	entity, err := Refetch(req.Ctx(), req.Scope, id, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return Respond(res, "collection", entity)
}
