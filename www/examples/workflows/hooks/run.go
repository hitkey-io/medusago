package hooks

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// region: run
// POST runs createProductsWorkflow with additional data.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	product := dal.NewRecord()
	product.Set("title", "Shirt")
	// ...

	additionalData := dal.NewRecord()
	additionalData.Set("custom_field", "test")

	_, err := workflows.CreateProductsWorkflow.Run(req.Ctx(), req.Scope, workflows.CreateProductsInput{
		Products:       []*dal.Record{product},
		AdditionalData: additionalData,
	})
	return err
}

// endregion: run
