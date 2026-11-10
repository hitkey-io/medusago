package steps

import (
	"fmt"
	"strings"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/csv"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsproduct "github.com/hitkey-io/medusago/packages/core/utils/src/product"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// NormalizeCsvOutput is { create, update }.
type NormalizeCsvOutput struct {
	Create []*dal.Record
	Update []*dal.Record
}

// detectDelimiter mirrors detectDelimiter: the first of , ; | found in the
// header line.
func detectDelimiter(data string) string {
	header := strings.SplitN(data, "\n", 2)[0]
	for _, d := range []string{",", ";", "|"} {
		if strings.Contains(header, d) {
			return d
		}
	}
	return ","
}

func importRegions(ctx composer.StepExecutionContext) ([]utilsproduct.Region, error) {
	res, err := commonsteps.QueryOf(ctx.Container).Graph(ctx.Context, query.GraphInput{Entity: "region", Fields: []string{"id", "name", "currency_code"}}, joiner.Options{})
	if err != nil {
		return nil, err
	}
	var regions []utilsproduct.Region
	for _, r := range res.Data {
		rec, _ := r.(*dal.Record)
		id, _ := rec.Get("id")
		name, _ := rec.Get("name")
		code, _ := rec.Get("currency_code")
		regions = append(regions, utilsproduct.Region{ID: fmt.Sprint(id), Name: fmt.Sprint(name), CurrencyCode: fmt.Sprint(code)})
	}
	return regions, nil
}

// NormalizeCsvStep mirrors normalizeCsvStep (the legacy import).
var NormalizeCsvStep = composer.CreateStep("normalize-product-csv",
	func(ctx composer.StepExecutionContext, content string) (composer.StepResponse[NormalizeCsvOutput], error) {
		var zero composer.StepResponse[NormalizeCsvOutput]
		rows := csv.Csv2Json(content, detectDelimiter(content))
		regions, err := importRegions(ctx)
		if err != nil {
			return zero, err
		}
		pre := make([]utilsproduct.Row, len(rows))
		for i, r := range rows {
			if pre[i], err = utilsproduct.PreProcess(r, i+1); err != nil {
				return zero, err
			}
		}
		products, err := utilsproduct.NewCSVNormalizer(pre, &utilsproduct.Context{Regions: regions}).Process(0)
		if err != nil {
			return zero, err
		}
		out := NormalizeCsvOutput{Create: []*dal.Record{}, Update: []*dal.Record{}}
		for _, handle := range products.ToCreate.Keys() {
			p, _ := products.ToCreate.Get(handle)
			parsed, err := utilsproduct.CreateProduct.Parse(p)
			if err != nil {
				return zero, err
			}
			out.Create = append(out.Create, parsed.(*dal.Record))
		}
		for _, id := range products.ToUpdate.Keys() {
			p, _ := products.ToUpdate.Get(id)
			parsed, err := utilsproduct.UpdateProduct.Parse(p)
			if err != nil {
				return zero, err
			}
			out.Update = append(out.Update, parsed.(*dal.Record))
		}
		return composer.NewStepResponse(out), nil
	}, nil)
