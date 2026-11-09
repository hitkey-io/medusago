package producttypes

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// POST mirrors POST of route.ts.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	result, err := workflows.CreateProductTypesWorkflow.Run(req.Ctx(), req.Scope, workflows.CreateProductTypesInput{ProductTypes: []*dal.Record{body}})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	entity, err := Refetch(req.Ctx(), req.Scope, id, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return Respond(res, "product_type", entity)
}
