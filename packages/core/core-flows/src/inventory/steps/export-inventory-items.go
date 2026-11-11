package steps

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	filetypes "github.com/hitkey-io/medusago/packages/core/types/src/file"
	"github.com/hitkey-io/medusago/packages/core/utils/src/csv"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ExportInventoryItemsDTO mirrors InventoryWorkflow.ExportInventoryItemsDTO.
type ExportInventoryItemsDTO struct {
	Select    []string
	Filter    *dal.Record
	BatchSize any
}

// ExportInventoryItemsStepOutput is { id, filename }.
type ExportInventoryItemsStepOutput struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
}

// beautifyKey mirrors beautifyKey: snake_case -> "Title Case".
func beautifyKey(key string) string {
	parts := strings.Split(key, "_")
	for i, p := range parts {
		if p != "" {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

func normalizeQuantity(v any) any {
	if v == nil {
		return ""
	}
	if b, err := totals.NewBigNumber(v); err == nil {
		return b.Numeric()
	}
	return v
}

func recGet(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func recordsOf(v any) []*dal.Record {
	var out []*dal.Record
	switch t := v.(type) {
	case []any:
		for _, x := range t {
			r, _ := x.(*dal.Record)
			out = append(out, r)
		}
	case []*dal.Record:
		out = t
	}
	return out
}

func orEmptyString(v any) any {
	if v == nil {
		return ""
	}
	return v
}

// normalizeInventoryForExport mirrors normalizeForExport.
func normalizeInventoryForExport(items []any, locations []*dal.Record) []any {
	out := []any{}
	for _, it := range items {
		item, _ := it.(*dal.Record)
		levels := map[any]*dal.Record{}
		for _, l := range recordsOf(recGet(item, "location_levels")) {
			if l != nil {
				levels[recGet(l, "location_id")] = l
			}
		}
		itemFields := dal.NewRecord()
		for _, k := range item.Keys() {
			v, _ := item.Get(k)
			itemFields.Set(beautifyKey("item_"+k), v)
		}
		itemFields.Delete("Item Location Levels")
		itemFields.Delete("Item Metadata")
		itemFields.Delete("Item Variants")
		locationColumns := dal.NewRecord()
		for _, loc := range locations {
			level := levels[recGet(loc, "id")]
			col := func(field string) string {
				return fmt.Sprintf("Location [%v] %s", recGet(loc, "name"), beautifyKey(field))
			}
			locationColumns.Set(col("stocked_quantity"), normalizeQuantity(recGet(level, "stocked_quantity")))
			locationColumns.Set(col("reserved_quantity"), normalizeQuantity(recGet(level, "reserved_quantity")))
			locationColumns.Set(col("available_quantity"), normalizeQuantity(recGet(level, "available_quantity")))
		}
		var variants []*dal.Record
		for _, v := range recordsOf(recGet(item, "variants")) {
			if v != nil {
				variants = append(variants, v)
			}
		}
		if len(variants) == 0 {
			variants = []*dal.Record{nil}
		}
		for _, variant := range variants {
			row := dal.NewRecord()
			for _, k := range itemFields.Keys() {
				v, _ := itemFields.Get(k)
				row.Set(k, v)
			}
			row.Set("Variant Id", orEmptyString(recGet(variant, "id")))
			row.Set("Variant Title", orEmptyString(recGet(variant, "title")))
			row.Set("Variant Barcode", orEmptyString(recGet(variant, "barcode")))
			row.Set("Variant Ean", orEmptyString(recGet(variant, "ean")))
			row.Set("Variant Upc", orEmptyString(recGet(variant, "upc")))
			for _, k := range locationColumns.Keys() {
				v, _ := locationColumns.Get(k)
				row.Set(k, v)
			}
			out = append(out, row)
		}
	}
	return out
}

// localeCompare approximates String.prototype.localeCompare for names.
func localeCompare(a, b string) int {
	if c := strings.Compare(strings.ToLower(a), strings.ToLower(b)); c != 0 {
		return c
	}
	return -strings.Compare(a, b)
}

// ExportInventoryItemsStep mirrors exportInventoryItemsStep.
var ExportInventoryItemsStep = composer.CreateStep("export-inventory-items",
	func(ctx composer.StepExecutionContext, in ExportInventoryItemsDTO) (composer.StepResponse[ExportInventoryItemsStepOutput], error) {
		var zero composer.StepResponse[ExportInventoryItemsStepOutput]
		q := commonsteps.QueryOf(ctx.Container)
		locRes, err := q.Graph(ctx.Context, query.GraphInput{Entity: "stock_location", Fields: []string{"id", "name"}}, joiner.Options{})
		if err != nil {
			return zero, err
		}
		locations := recordsOf(locRes.Data)
		sort.SliceStable(locations, func(i, j int) bool {
			return localeCompare(fmt.Sprint(recGet(locations[i], "name")), fmt.Sprint(recGet(locations[j], "name"))) < 0
		})
		filename := fmt.Sprintf("%d-inventory-item-exports.csv", time.Now().UnixMilli())
		stream, err := commonsteps.FileStreamServiceOf(ctx.Container).GetUploadStream(ctx.Context, filetypes.ProviderUploadStreamDTO{Filename: filename, MimeType: "text/csv"})
		if err != nil {
			return zero, err
		}
		pageSize := commonsteps.BatchSize(in.BatchSize, 50)
		fields := commonsteps.Deduplicate(append([]string{"id", "sku", "variants.id", "variants.title", "variants.barcode", "variants.ean", "variants.upc"}, in.Select...))
		hasLevels := false
		for _, f := range fields {
			if strings.Contains(f, "location_levels") {
				hasLevels = true
			}
		}
		if !hasLevels {
			fields = append(fields, "location_levels.*")
		}
		hasHeader := false
		werr := func() (err error) {
			defer func() { err = errors.Join(err, stream.File.Close()) }()
			for page := 0; ; page++ {
				filters := dal.NewRecord()
				if in.Filter != nil {
					filters = in.Filter
				}
				pagination := dal.NewRecord()
				pagination.Set("skip", page*pageSize)
				pagination.Set("take", pageSize)
				res, err := q.Graph(ctx.Context, query.GraphInput{Entity: "inventory_items", Fields: fields, Filters: filters, Pagination: pagination}, joiner.Options{})
				if err != nil {
					return err
				}
				if len(res.Data) == 0 {
					break
				}
				chunk := csv.Json2Csv(normalizeInventoryForExport(res.Data, locations), commonsteps.ExportCsvOptions(!hasHeader))
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
		return composer.NewStepResponse(ExportInventoryItemsStepOutput{ID: stream.FileKey, Filename: filename}, any(stream.FileKey)), nil
	},
	func(ctx composer.StepExecutionContext, input any) error {
		id, _ := input.(string)
		if id == "" {
			return nil
		}
		return commonsteps.FileStreamServiceOf(ctx.Container).DeleteFiles(ctx.Context, []string{id})
	})
