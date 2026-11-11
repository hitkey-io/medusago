package steps

import (
	"errors"
	"fmt"
	"time"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	orderutils "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	filetypes "github.com/hitkey-io/medusago/packages/core/types/src/file"
	"github.com/hitkey-io/medusago/packages/core/utils/src/csv"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ExportOrdersStepInput mirrors ExportOrdersStepInput.
type ExportOrdersStepInput struct {
	BatchSize any
	Select    []string
	Filter    *dal.Record
}

// ExportOrdersStepOutput mirrors ExportOrdersStepOutput.
type ExportOrdersStepOutput struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
}

func orEmpty(v any) any {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		if t == "" {
			return ""
		}
	case bool:
		if !t {
			return ""
		}
	case float64:
		if t == 0 || t != t {
			return ""
		}
	}
	return v
}

func setDefined(r *dal.Record, key string, src *dal.Record, from string) {
	if v, ok := src.Get(from); ok {
		r.Set(key, v)
	}
}

func normalizeOrderForExport(order *dal.Record) any {
	customer, _ := get(order, "customer").(*dal.Record)
	if customer == nil {
		customer = dal.NewRecord()
	}
	shipping, _ := get(order, "shipping_address").(*dal.Record)
	if shipping == nil {
		shipping = dal.NewRecord()
	}
	r := dal.NewRecord()
	setDefined(r, "Order_ID", order, "id")
	setDefined(r, "Display_ID", order, "display_id")
	setDefined(r, "Order status", order, "status")
	setDefined(r, "Date", order, "created_at")
	r.Set("Customer First name", orEmpty(get(customer, "first_name")))
	r.Set("Customer Last name", orEmpty(get(customer, "last_name")))
	r.Set("Customer Email", orEmpty(get(customer, "email")))
	r.Set("Customer ID", orEmpty(get(customer, "id")))
	r.Set("Shipping Address 1", orEmpty(get(shipping, "address_1")))
	r.Set("Shipping Address 2", orEmpty(get(shipping, "address_2")))
	r.Set("Shipping Country Code", orEmpty(get(shipping, "country_code")))
	r.Set("Shipping City", orEmpty(get(shipping, "city")))
	r.Set("Shipping Postal Code", orEmpty(get(shipping, "postal_code")))
	setDefined(r, "Shipping Region ID", order, "region_id")
	setDefined(r, "Fulfillment Status", order, "fulfillment_status")
	setDefined(r, "Payment Status", order, "payment_status")
	setDefined(r, "Subtotal", order, "subtotal")
	setDefined(r, "Shipping Total", order, "shipping_total")
	setDefined(r, "Discount Total", order, "discount_total")
	setDefined(r, "Gift Card Total", order, "gift_card_total")
	setDefined(r, "Refunded Total", order, "refunded_total")
	setDefined(r, "Tax Total", order, "tax_total")
	setDefined(r, "Total", order, "total")
	setDefined(r, "Currency Code", order, "currency_code")
	return commonsteps.JSONRoundTrip(r)
}

// ExportOrdersStep mirrors exportOrdersStep.
var ExportOrdersStep = composer.CreateStep("export-orders",
	func(ctx composer.StepExecutionContext, in ExportOrdersStepInput) (composer.StepResponse[ExportOrdersStepOutput], error) {
		var zero composer.StepResponse[ExportOrdersStepOutput]
		files := commonsteps.FileStreamServiceOf(ctx.Container)
		filename := fmt.Sprintf("%d-order-exports.csv", time.Now().UnixMilli())
		stream, err := files.GetUploadStream(ctx.Context, filetypes.ProviderUploadStreamDTO{Filename: filename, MimeType: "text/csv"})
		if err != nil {
			return zero, err
		}
		pageSize := commonsteps.BatchSize(in.BatchSize, 50)
		fields := commonsteps.Deduplicate(append(append([]string{}, in.Select...), "id", "status", "items.*", "customer.*", "shipping_address.*",
			"payment_collections.status", "payment_collections.amount", "payment_collections.captured_amount", "payment_collections.refunded_amount",
			"fulfillments.packed_at", "fulfillments.shipped_at", "fulfillments.delivered_at", "fulfillments.canceled_at"))
		hasHeader := false
		werr := func() (err error) {
			defer func() { err = errors.Join(err, stream.File.Close()) }()
			for page := 0; ; page++ {
				filters := dal.NewRecord()
				if in.Filter != nil {
					for _, k := range in.Filter.Keys() {
						v, _ := in.Filter.Get(k)
						filters.Set(k, v)
					}
				}
				ne := dal.NewRecord()
				ne.Set("$ne", "draft")
				filters.Set("status", ne)
				pagination := dal.NewRecord()
				pagination.Set("skip", page*pageSize)
				pagination.Set("take", pageSize)
				res, err := commonsteps.QueryOf(ctx.Container).Graph(ctx.Context, query.GraphInput{Entity: "order", Filters: filters, Pagination: pagination, Fields: fields}, joiner.Options{})
				if err != nil {
					return err
				}
				if len(res.Data) == 0 {
					break
				}
				rows := make([]any, len(res.Data))
				for i, o := range res.Data {
					order, _ := o.(*dal.Record)
					order.Set("payment_status", orderutils.GetLastPaymentStatus(order))
					order.Set("fulfillment_status", orderutils.GetLastFulfillmentStatus(order))
					order.Delete("version")
					order.Delete("payment_collections")
					order.Delete("fulfillments")
					rows[i] = normalizeOrderForExport(order)
				}
				chunk := csv.Json2Csv(rows, commonsteps.ExportCsvOptions(!hasHeader))
				if hasHeader {
					chunk = "\n" + chunk
				}
				if _, err := stream.File.WriteString(chunk); err != nil {
					return err
				}
				hasHeader = true
				if len(res.Data) < pageSize {
					break
				}
			}
			return nil
		}()
		if werr != nil {
			return zero, werr
		}
		return composer.NewStepResponse(ExportOrdersStepOutput{ID: stream.FileKey, Filename: filename}, any(stream.FileKey)), nil
	},
	func(ctx composer.StepExecutionContext, input any) error {
		id, _ := input.(string)
		if id == "" {
			return nil
		}
		return commonsteps.FileStreamServiceOf(ctx.Container).DeleteFiles(ctx.Context, []string{id})
	})
