package steps

import (
	"fmt"
	"time"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/helpers"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/csv"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ExportProductsDTO mirrors ProductWorkflow.ExportProductsDTO.
type ExportProductsDTO struct {
	Select    []string
	Filter    *dal.Record
	BatchSize any
}

// ExportProductsStepOutput is { id, filename }.
type ExportProductsStepOutput struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
}

func recordList(v []any) []*dal.Record {
	out := make([]*dal.Record, 0, len(v))
	for _, x := range v {
		if r, ok := x.(*dal.Record); ok {
			out = append(out, r)
		}
	}
	return out
}

// ExportProductsStep mirrors exportProductsStep.
var ExportProductsStep = composer.CreateStep("export-products",
	func(ctx composer.StepExecutionContext, in ExportProductsDTO) (composer.StepResponse[ExportProductsStepOutput], error) {
		var zero composer.StepResponse[ExportProductsStepOutput]
		q := commonsteps.QueryOf(ctx.Container)
		regionRes, err := q.Graph(ctx.Context, query.GraphInput{Entity: "region", Fields: []string{"id", "name", "currency_code"}}, joiner.Options{})
		if err != nil {
			return zero, err
		}
		regions := recordList(regionRes.Data)
		filename := fmt.Sprintf("%d-product-exports.csv", time.Now().UnixMilli())
		stream, err := commonsteps.FileStreamServiceOf(ctx.Container).GetUploadStream(ctx.Context, filename, "")
		if err != nil {
			return zero, err
		}
		pageSize := commonsteps.BatchSize(in.BatchSize, 50)
		fields := commonsteps.Deduplicate(append([]string{"id", "handle"}, in.Select...))
		filters := dal.NewRecord()
		var salesChannelID any
		if in.Filter != nil {
			for _, k := range in.Filter.Keys() {
				v, _ := in.Filter.Get(k)
				if k == "sales_channel_id" {
					salesChannelID = v
					continue
				}
				filters.Set(k, v)
			}
		}
		hasSalesChannel := salesChannelID != nil && salesChannelID != "" && salesChannelID != false
		getProducts := func(page int) ([]any, error) {
			if hasSalesChannel {
				f := dal.NewRecord()
				f.Set("sales_channel_id", salesChannelID)
				p := dal.NewRecord()
				p.Set("skip", page*pageSize)
				p.Set("take", pageSize)
				order := dal.NewRecord()
				order.Set("product_id", "ASC")
				p.Set("order", order)
				res, err := q.Graph(ctx.Context, query.GraphInput{Entity: "product_sales_channel", Filters: f, Fields: []string{"product_id"}, Pagination: p}, joiner.Options{})
				if err != nil {
					return nil, err
				}
				ids := []any{}
				for _, r := range recordList(res.Data) {
					id, _ := r.Get("product_id")
					ids = append(ids, id)
				}
				filters.Set("id", ids)
			}
			var pagination *dal.Record
			if !hasSalesChannel {
				pagination = dal.NewRecord()
				pagination.Set("skip", page*pageSize)
				pagination.Set("take", pageSize)
				order := dal.NewRecord()
				order.Set("id", "ASC")
				pagination.Set("order", order)
			}
			res, err := q.Graph(ctx.Context, query.GraphInput{Entity: "product", Fields: fields, Filters: filters, Pagination: pagination}, joiner.Options{})
			if err != nil {
				return nil, err
			}
			return res.Data, nil
		}

		var exportKeys []string
		seen := map[string]bool{}
		var allIDs []any
		for page := 0; ; page++ {
			products, err := getProducts(page)
			if err != nil {
				return zero, err
			}
			if len(products) == 0 {
				break
			}
			normalized, err := helpers.NormalizeForExport(products, regions)
			if err != nil {
				return zero, err
			}
			for _, p := range recordList(products) {
				id, _ := p.Get("id")
				allIDs = append(allIDs, id)
			}
			helpers.AppendProductExportKeys(normalized, &exportKeys, seen)
			if len(products) < pageSize {
				break
			}
		}
		if exportKeys == nil {
			exportKeys = []string{}
		}

		hasHeader := false
		werr := func() error {
			defer stream.File.Close()
			for i := 0; i < len(allIDs); i += pageSize {
				f := dal.NewRecord()
				f.Set("id", allIDs[i:min(i+pageSize, len(allIDs))])
				res, err := q.Graph(ctx.Context, query.GraphInput{Entity: "product", Fields: fields, Filters: f}, joiner.Options{})
				if err != nil {
					return err
				}
				normalized, err := helpers.NormalizeForExport(res.Data, regions)
				if err != nil {
					return err
				}
				opts := commonsteps.ExportCsvOptions(!hasHeader)
				opts.Keys = exportKeys
				chunk := csv.Json2Csv(normalized, opts)
				if hasHeader {
					chunk = "\n" + chunk
				}
				if _, err := stream.File.WriteString(chunk); err != nil {
					return err
				}
				hasHeader = true
			}
			return nil
		}()
		if werr != nil {
			return zero, werr
		}
		return composer.NewStepResponse(ExportProductsStepOutput{ID: stream.FileKey, Filename: filename}, any(stream.FileKey)), nil
	},
	func(ctx composer.StepExecutionContext, input any) error {
		id, _ := input.(string)
		if id == "" {
			return nil
		}
		return commonsteps.FileStreamServiceOf(ctx.Container).DeleteFiles(ctx.Context, []string{id})
	})
