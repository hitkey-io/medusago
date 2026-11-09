// Package workflows mirrors @medusajs/core-flows/src/file/workflows.
package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/file/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	localfile "github.com/hitkey-io/medusago/packages/modules/providers/file-local/src/services"
)

// UploadFilesWorkflow mirrors uploadFilesWorkflow.
var UploadFilesWorkflow = composer.CreateWorkflow("upload-files",
	func(e *composer.Execution, files []*localfile.UploadFile) ([]*dal.Record, error) {
		return steps.UploadFilesStep.Run(e, files)
	})

// DeleteFilesWorkflow mirrors deleteFilesWorkflow.
var DeleteFilesWorkflow = composer.CreateWorkflow("delete-files",
	func(e *composer.Execution, ids []any) (any, error) {
		_, err := steps.DeleteFilesStep.Run(e, ids)
		return nil, err
	})
