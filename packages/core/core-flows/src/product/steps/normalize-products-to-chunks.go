package steps

import (
	"context"
	"errors"
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/csv"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	utilsproduct "github.com/hitkey-io/medusago/packages/core/utils/src/product"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	localfile "github.com/hitkey-io/medusago/packages/modules/providers/file-local/src/services"
)

// ImportChunk mirrors Chunk ({ id, toCreate, toUpdate }).
type ImportChunk struct {
	ID       string `json:"id"`
	ToCreate int    `json:"toCreate"`
	ToUpdate int    `json:"toUpdate"`
}

// ImportSummary mirrors ImportProductsSummary.
type ImportSummary struct {
	ToCreate int `json:"toCreate"`
	ToUpdate int `json:"toUpdate"`
}

// NormalizeCsvToChunksOutput is { chunks, summary }.
type NormalizeCsvToChunksOutput struct {
	Chunks  []ImportChunk `json:"chunks"`
	Summary ImportSummary `json:"summary"`
}

// ImportFileService is the part of the file module the import uses.
type ImportFileService interface {
	GetAsBuffer(ctx context.Context, id string) ([]byte, error)
	CreateFiles(ctx context.Context, files []*localfile.UploadFile) ([]*dal.Record, error)
	DeleteFiles(ctx context.Context, ids []string) error
}

// ImportFileServiceOf resolves the file module.
func ImportFileServiceOf(c composer.Container) ImportFileService {
	s, _ := c.Resolve(utilsmodulessdk.FILE).(ImportFileService)
	return s
}

func processChunk(ctx context.Context, files ImportFileService, fileKey string, rows []utilsproduct.Row, currentRow int, regions []utilsproduct.Region) (ImportChunk, error) {
	products, err := utilsproduct.NewCSVNormalizer(rows, &utilsproduct.Context{Regions: regions}).Process(currentRow)
	if err != nil {
		return ImportChunk{}, err
	}
	create := []any{}
	for _, handle := range products.ToCreate.Keys() {
		p, _ := products.ToCreate.Get(handle)
		parsed, err := utilsproduct.CreateProduct.Parse(p)
		if err != nil {
			return ImportChunk{}, err
		}
		create = append(create, parsed)
	}
	update := []any{}
	for _, id := range products.ToUpdate.Keys() {
		p, _ := products.ToUpdate.Get(id)
		parsed, err := utilsproduct.UpdateProduct.Parse(p)
		if err != nil {
			return ImportChunk{}, err
		}
		update = append(update, parsed)
	}
	payload := dal.NewRecord()
	payload.Set("create", create)
	payload.Set("update", update)
	content, err := dal.MarshalValue(payload)
	if err != nil {
		return ImportChunk{}, err
	}
	created, err := files.CreateFiles(ctx, []*localfile.UploadFile{{Filename: fileKey + ".json", Content: string(content), MimeType: "application/json"}})
	if err != nil {
		return ImportChunk{}, err
	}
	id, _ := created[0].Get("id")
	return ImportChunk{ID: fmt.Sprint(id), ToCreate: len(create), ToUpdate: len(update)}, nil
}

func createChunks(ctx context.Context, files ImportFileService, fileKey string, records []*dal.Record, regions []utilsproduct.Region) ([]ImportChunk, error) {
	const rowsToRead = 1000
	currentRow, readSoFar := 0, 0
	var chunks []ImportChunk
	var rows []utilsproduct.Row
	var currentUnique any
	fail := func(err error) ([]ImportChunk, error) {
		ids := make([]string, len(chunks))
		for i, c := range chunks {
			ids[i] = c.ID
		}
		_ = files.DeleteFiles(ctx, ids)
		return nil, err
	}
	for _, row := range records {
		readSoFar++
		currentRow++
		normalized, err := utilsproduct.PreProcess(row, currentRow)
		if err != nil {
			return fail(err)
		}
		unique, _ := normalized.Get("product id")
		if s, _ := unique.(string); s == "" {
			unique, _ = normalized.Get("product handle")
		}
		if readSoFar > rowsToRead {
			if unique != currentUnique {
				chunk, err := processChunk(ctx, files, fmt.Sprintf("%s-%d", fileKey, len(chunks)+1), rows, currentRow, regions)
				if err != nil {
					return fail(err)
				}
				chunks = append(chunks, chunk)
				rows = []utilsproduct.Row{normalized}
				readSoFar = 0
			} else {
				rows = append(rows, normalized)
			}
		} else {
			rows = append(rows, normalized)
		}
		currentUnique = unique
	}
	if len(rows) > 0 {
		chunk, err := processChunk(ctx, files, fmt.Sprintf("%s-%d", fileKey, len(chunks)+1), rows, currentRow, regions)
		if err != nil {
			return fail(err)
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

// NormalizeCsvToChunksStep mirrors normalizeCsvToChunksStep.
var NormalizeCsvToChunksStep = composer.CreateStep("normalize-product-csv-to-chunks",
	func(ctx composer.StepExecutionContext, fileKey string) (composer.StepResponse[NormalizeCsvToChunksOutput], error) {
		var zero composer.StepResponse[NormalizeCsvToChunksOutput]
		files := ImportFileServiceOf(ctx.Container)
		res, err := commonsteps.QueryOf(ctx.Container).Graph(ctx.Context, query.GraphInput{Entity: "region", Fields: []string{"id", "name", "currency_code"}}, joiner.Options{})
		if err != nil {
			return zero, err
		}
		var regions []utilsproduct.Region
		for _, r := range res.Data {
			rec, _ := r.(*dal.Record)
			id, _ := rec.Get("id")
			name, _ := rec.Get("name")
			code, _ := rec.Get("currency_code")
			regions = append(regions, utilsproduct.Region{ID: fmt.Sprint(id), Name: fmt.Sprint(name), CurrencyCode: fmt.Sprint(code)})
		}
		content, err := files.GetAsBuffer(ctx.Context, fileKey)
		if err != nil {
			return zero, err
		}
		records, err := csv.ParseColumns(string(content))
		if err != nil {
			var ce *csv.CsvError
			if errors.As(err, &ce) {
				return zero, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, ce.Message)
			}
			return zero, err
		}
		chunks, err := createChunks(ctx.Context, files, fileKey, records, regions)
		if err != nil {
			return zero, err
		}
		if chunks == nil {
			chunks = []ImportChunk{}
		}
		out := NormalizeCsvToChunksOutput{Chunks: chunks}
		for _, c := range chunks {
			out.Summary.ToCreate += c.ToCreate
			out.Summary.ToUpdate += c.ToUpdate
		}
		return composer.NewStepResponse(out), nil
	}, nil)

// WaitConfirmationProductImportStepID mirrors waitConfirmationProductImportStepId.
const WaitConfirmationProductImportStepID = "wait-confirmation-product-import"

// WaitConfirmationProductImportStep mirrors waitConfirmationProductImportStep:
// an async step completed by the confirm route.
var WaitConfirmationProductImportStep = composer.CreateStep(WaitConfirmationProductImportStepID,
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[any], error) {
		return composer.NewStepResponse[any](nil), nil
	}, nil).Config("", composer.StepOptions{Async: true, Timeout: 60 * 60})
