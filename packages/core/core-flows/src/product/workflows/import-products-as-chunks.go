package workflows

import (
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	notificationsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/notification/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func recordsOfAny(v any) []*dal.Record {
	l, _ := v.([]any)
	out := make([]*dal.Record, 0, len(l))
	for _, x := range l {
		if r, ok := x.(*dal.Record); ok {
			out = append(out, r)
		}
	}
	return out
}

// ProcessImportChunksStep mirrors processImportChunksStep (async: after
// importing the chunks it waits for setStepSuccess, like upstream). It
// lives here because it runs batchProductsWorkflow.
var ProcessImportChunksStep = composer.CreateStep("process-import-chunks",
	func(ctx composer.StepExecutionContext, in steps.NormalizeCsvToChunksOutput) (composer.StepResponse[*dal.Record], error) {
		files := steps.ImportFileServiceOf(ctx.Container)
		ids := make([]string, len(in.Chunks))
		for i, c := range in.Chunks {
			ids[i] = c.ID
		}
		defer func() { _ = files.DeleteFiles(ctx.Context, ids) }()
		for _, chunk := range in.Chunks {
			content, err := files.GetAsBuffer(ctx.Context, chunk.ID)
			if err != nil {
				return composer.StepResponse[*dal.Record]{}, err
			}
			parsed, err := dal.DecodeOrderedJSON(content)
			if err != nil {
				return composer.StepResponse[*dal.Record]{}, err
			}
			rec, _ := parsed.(*dal.Record)
			create, _ := rec.Get("create")
			update, _ := rec.Get("update")
			input := BatchProductsInput{Create: recordsOfAny(create), Update: recordsOfAny(update)}
			if _, err := BatchProductsWorkflow.Run(ctx.Context, ctx.Container, input); err != nil {
				return composer.StepResponse[*dal.Record]{}, err
			}
		}
		out := dal.NewRecord()
		out.Set("completed", true)
		return composer.NewStepResponse(out), nil
	}, nil).Config("", composer.StepOptions{Async: true})

// ImportProductsAsChunksInput is { fileKey, filename }.
type ImportProductsAsChunksInput struct {
	FileKey  string `json:"fileKey"`
	Filename string `json:"filename"`
}

// ImportProductsAsChunksWorkflowID mirrors importProductsAsChunksWorkflowId.
const ImportProductsAsChunksWorkflowID = "import-products-as-chunks"

// ImportProductsAsChunksWorkflow mirrors importProductsAsChunksWorkflow: run()
// returns the summary once the file is normalized; the import waits for the
// confirmation.
var ImportProductsAsChunksWorkflow = composer.CreateWorkflow(ImportProductsAsChunksWorkflowID,
	func(e *composer.Execution, in ImportProductsAsChunksInput) (steps.ImportSummary, error) {
		batch, err := steps.NormalizeCsvToChunksStep.Run(e, in.FileKey)
		if err != nil {
			return steps.ImportSummary{}, err
		}
		composer.SetPartialResult(e, batch.Summary)
		if _, err := steps.WaitConfirmationProductImportStep.Run(e, nil); err != nil {
			return steps.ImportSummary{}, err
		}
		failure := []*dal.Record{commonsteps.FeedNotification("", commonsteps.ExportFeedData("Product import", fmt.Sprintf("Failed to import products from file %s", in.Filename), nil))}
		if _, err := notificationsteps.NotifyOnFailureStep.Run(e, failure); err != nil {
			return steps.ImportSummary{}, err
		}
		if _, err := ProcessImportChunksStep.Run(e, batch); err != nil {
			return steps.ImportSummary{}, err
		}
		done := []*dal.Record{commonsteps.FeedNotification("", commonsteps.ExportFeedData("Product import", fmt.Sprintf("Product import of file %s completed successfully!", in.Filename), nil))}
		if _, err := notificationsteps.SendNotificationsStep.Run(e, done); err != nil {
			return steps.ImportSummary{}, err
		}
		return batch.Summary, nil
	})
