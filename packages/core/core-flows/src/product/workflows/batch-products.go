package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// BatchProductsInput mirrors BatchProductWorkflowInput.
type BatchProductsInput struct {
	Create []*dal.Record
	Update []*dal.Record
	Delete []any
}

// BatchProductsOutput mirrors BatchWorkflowOutput<ProductDTO>.
type BatchProductsOutput struct {
	Created []*dal.Record `json:"created"`
	Updated []*dal.Record `json:"updated"`
	Deleted []any         `json:"deleted"`
}

// BatchProductsWorkflow mirrors batchProductsWorkflow.
var BatchProductsWorkflow = composer.CreateWorkflow("batch-products",
	func(e *composer.Execution, input BatchProductsInput) (BatchProductsOutput, error) {
		update := input.Update
		if update == nil {
			update = []*dal.Record{}
		}
		processed, err := steps.ProcessProductOptionsForImportStep.Run(e, update)
		if err != nil {
			return BatchProductsOutput{}, err
		}
		out := BatchProductsOutput{Created: []*dal.Record{}, Updated: []*dal.Record{}, Deleted: input.Delete}
		if out.Deleted == nil {
			out.Deleted = []any{}
		}
		err = composer.Parallelize(
			func() error {
				if len(input.Create) == 0 {
					return nil
				}
				r, err := CreateProductsWorkflow.RunAsStep(e, CreateProductsInput{Products: input.Create})
				if r != nil {
					out.Created = r
				}
				return err
			},
			func() error {
				if len(processed) == 0 {
					return nil
				}
				r, err := UpdateProductsWorkflow.RunAsStep(e, UpdateProductsInput{Batch: true, Products: processed})
				if r != nil {
					out.Updated = r
				}
				return err
			},
			func() error {
				if len(input.Delete) == 0 {
					return nil
				}
				_, err := DeleteProductsWorkflow.RunAsStep(e, DeleteProductsInput{IDs: input.Delete})
				return err
			},
		)
		return out, err
	})
