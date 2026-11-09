package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// BatchProductVariantsInput mirrors BatchProductVariantsWorkflowInput.
type BatchProductVariantsInput struct {
	Create []*dal.Record
	Update []*dal.Record
	Delete []any
}

// BatchProductVariantsOutput mirrors BatchProductVariantsWorkflowOutput.
type BatchProductVariantsOutput struct {
	Created []*dal.Record
	Updated []*dal.Record
	Deleted []any
}

// BatchProductVariantsWorkflow mirrors batchProductVariantsWorkflow.
var BatchProductVariantsWorkflow = composer.CreateWorkflow("batch-product-variants",
	func(e *composer.Execution, input BatchProductVariantsInput) (BatchProductVariantsOutput, error) {
		create := input.Create
		if create == nil {
			create = []*dal.Record{}
		}
		update := input.Update
		if update == nil {
			update = []*dal.Record{}
		}
		del := input.Delete
		if del == nil {
			del = []any{}
		}
		var out BatchProductVariantsOutput
		err := composer.Parallelize(
			func() error {
				r, err := CreateProductVariantsWorkflow.RunAsStep(e, CreateProductVariantsInput{ProductVariants: create})
				out.Created = r
				return err
			},
			func() error {
				r, err := UpdateProductVariantsWorkflow.RunAsStep(e, UpdateProductVariantsInput{Batch: true, ProductVariants: update})
				out.Updated = r
				return err
			},
			func() error {
				_, err := DeleteProductVariantsWorkflow.RunAsStep(e, DeleteProductVariantsInput{IDs: del})
				return err
			},
		)
		out.Deleted = del
		return out, err
	})
