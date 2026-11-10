package steps

import (
	"fmt"
	"strings"
	"time"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/csv"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CollectCustomerIdsToExportStepInput mirrors CollectCustomerIdsToExportStepInput.
type CollectCustomerIdsToExportStepInput struct {
	Filter    *dal.Record
	BatchSize any
}

// CollectCustomerIdsToExportStep mirrors collectCustomerIdsToExportStep.
var CollectCustomerIdsToExportStep = composer.CreateStep("collect-customer-ids-to-export",
	func(ctx composer.StepExecutionContext, in CollectCustomerIdsToExportStepInput) (composer.StepResponse[[]string], error) {
		pageSize := commonsteps.BatchSize(in.BatchSize, 200)
		ids := []string{}
		for page := 0; ; page++ {
			filters := dal.NewRecord()
			if in.Filter != nil {
				for _, k := range in.Filter.Keys() {
					v, _ := in.Filter.Get(k)
					filters.Set(k, v)
				}
			}
			pagination := dal.NewRecord()
			pagination.Set("skip", page*pageSize)
			pagination.Set("take", pageSize)
			res, err := commonsteps.QueryOf(ctx.Container).Graph(ctx.Context, query.GraphInput{Entity: "customer", Filters: filters, Pagination: pagination, Fields: []string{"id"}}, joiner.Options{})
			if err != nil {
				return composer.StepResponse[[]string]{}, err
			}
			if len(res.Data) == 0 {
				break
			}
			for _, c := range res.Data {
				r, _ := c.(*dal.Record)
				id, _ := r.Get("id")
				ids = append(ids, fmt.Sprint(id))
			}
			if len(res.Data) < pageSize {
				break
			}
		}
		return composer.NewStepResponse(ids), nil
	}, nil)

// ExportCustomersStepInput mirrors ExportCustomersStepInput.
type ExportCustomersStepInput struct {
	Select         []string
	CustomerIDs    []string
	BatchSize      any
	Format         string
	AdditionalData *dal.Record
}

// ExportCustomersStepOutput mirrors ExportCustomersStepOutput.
type ExportCustomersStepOutput struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
}

var exportCustomersDefaultFields = []string{"id", "email", "first_name", "last_name", "company_name", "phone", "has_account", "created_by", "created_at", "updated_at", "metadata",
	"addresses.*", "groups.id", "groups.name", "orders.id", "orders.display_id", "orders.status", "orders.email", "orders.currency_code", "orders.total", "orders.created_at"}

// normalizeCustomerForExport mirrors normalizeCustomerForExport.
func normalizeCustomerForExport(customer *dal.Record, additional any) any {
	record := dal.NewRecord()
	for _, k := range customer.Keys() {
		v, _ := customer.Get(k)
		record.Set(k, v)
	}
	if extra, ok := additional.(*dal.Record); ok {
		for _, k := range extra.Keys() {
			v, _ := extra.Get(k)
			record.Set(k, v)
		}
	}
	return commonsteps.JSONRoundTrip(record)
}

// ExportCustomersStep mirrors exportCustomersStep.
var ExportCustomersStep = composer.CreateStep("export-customers",
	func(ctx composer.StepExecutionContext, in ExportCustomersStepInput) (composer.StepResponse[ExportCustomersStepOutput], error) {
		var zero composer.StepResponse[ExportCustomersStepOutput]
		format := "json"
		if in.Format == "csv" {
			format = "csv"
		}
		filename := fmt.Sprintf("%d-customer-exports.%s", time.Now().UnixMilli(), format)
		stream, err := commonsteps.FileStreamServiceOf(ctx.Container).GetUploadStream(ctx.Context, filename, "")
		if err != nil {
			return zero, err
		}
		pageSize := commonsteps.BatchSize(in.BatchSize, 50)
		fields := commonsteps.Deduplicate(append(append([]string{}, exportCustomersDefaultFields...), in.Select...))
		hasHeader, hasWrittenRecord := false, false
		werr := func() error {
			defer stream.File.Close()
			write := func(s string) error {
				_, err := stream.File.WriteString(s)
				return err
			}
			if format == "json" {
				if err := write("["); err != nil {
					return err
				}
			}
			for i := 0; i < len(in.CustomerIDs); i += pageSize {
				end := min(i+pageSize, len(in.CustomerIDs))
				ids := make([]any, 0, end-i)
				for _, id := range in.CustomerIDs[i:end] {
					ids = append(ids, id)
				}
				filters := dal.NewRecord()
				filters.Set("id", ids)
				res, err := commonsteps.QueryOf(ctx.Container).Graph(ctx.Context, query.GraphInput{Entity: "customer", Filters: filters, Fields: fields}, joiner.Options{})
				if err != nil {
					return err
				}
				if len(res.Data) == 0 {
					continue
				}
				records := make([]any, len(res.Data))
				for j, c := range res.Data {
					customer, _ := c.(*dal.Record)
					var extra any
					if in.AdditionalData != nil {
						id, _ := customer.Get("id")
						extra, _ = in.AdditionalData.Get(fmt.Sprint(id))
					}
					records[j] = normalizeCustomerForExport(customer, extra)
				}
				if format == "json" {
					parts := make([]string, len(records))
					for j, r := range records {
						b, err := dal.MarshalValue(r)
						if err != nil {
							return err
						}
						parts[j] = string(b)
					}
					prefix := ""
					if hasWrittenRecord {
						prefix = ","
					}
					if err := write(prefix + strings.Join(parts, ",")); err != nil {
						return err
					}
					hasWrittenRecord = true
				} else {
					chunk := csv.Json2Csv(records, commonsteps.ExportCsvOptions(!hasHeader))
					if hasHeader {
						chunk = "\n" + chunk
					}
					if err := write(chunk); err != nil {
						return err
					}
					hasHeader = true
				}
			}
			if format == "json" {
				return write("]")
			}
			return nil
		}()
		if werr != nil {
			return zero, werr
		}
		return composer.NewStepResponse(ExportCustomersStepOutput{ID: stream.FileKey, Filename: filename}, any(stream.FileKey)), nil
	},
	func(ctx composer.StepExecutionContext, input any) error {
		id, _ := input.(string)
		if id == "" {
			return nil
		}
		return commonsteps.FileStreamServiceOf(ctx.Container).DeleteFiles(ctx.Context, []string{id})
	})
