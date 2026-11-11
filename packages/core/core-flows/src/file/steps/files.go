// Package steps mirrors @medusajs/core-flows/src/file/steps.
package steps

import (
	"fmt"

	filetypes "github.com/hitkey-io/medusago/packages/core/types/src/file"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	fileservices "github.com/hitkey-io/medusago/packages/modules/file/src/services"
)

func fileService(c composer.Container) *fileservices.FileModuleService {
	return c.Resolve(utilsmodulessdk.FILE).(*fileservices.FileModuleService)
}

func keys(ids []any) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = fmt.Sprint(id)
	}
	return out
}

// UploadFilesStep mirrors uploadFilesStep.
var UploadFilesStep = composer.CreateStep("upload-files",
	func(ctx composer.StepExecutionContext, files []*filetypes.ProviderUploadFileDTO) (composer.StepResponse[[]*dal.Record], error) {
		created, err := fileService(ctx.Container).CreateFiles(ctx.Context, files)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		ids := make([]any, len(created))
		for i, c := range created {
			ids[i], _ = c.Get("id")
		}
		return composer.NewStepResponse(created, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		return fileService(ctx.Container).DeleteFiles(ctx.Context, keys(ids))
	})

// DeleteFilesStep mirrors deleteFilesStep (no compensation).
var DeleteFilesStep = composer.CreateStep("delete-files",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		if err := fileService(ctx.Container).DeleteFiles(ctx.Context, keys(ids)); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)
