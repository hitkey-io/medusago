// Package calculate mirrors api/store/shipping-options/[id]/calculate.
package calculate

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	cartID, _ := body.Get("cart_id")
	so := dal.NewRecord()
	so.Set("id", req.Params["id"])
	if d, ok := body.Get("data"); ok {
		so.Set("data", d)
	}
	id, _ := cartID.(string)
	prices, err := workflows.CalculateShippingOptionsPricesWorkflow.Run(req.Ctx(), req.Scope, workflows.CalculateShippingOptionsPricesInput{
		CartID: id, ShippingOptions: []*dal.Record{so},
	})
	if err != nil {
		return err
	}
	f := dal.NewRecord()
	f.Set("id", req.Params["id"])
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: "shipping_option", Fields: req.QueryConfig.Fields, Filters: f}, joiner.Options{})
	if err != nil {
		return err
	}
	var option *dal.Record
	if len(r.Data) > 0 {
		option, _ = r.Data[0].(*dal.Record)
	}
	if option == nil {
		return fmt.Errorf("Cannot set properties of undefined (setting 'calculated_price')")
	}
	if len(prices) == 0 {
		return fmt.Errorf("Cannot read properties of undefined (reading 'calculated_amount')")
	}
	price := prices[0]
	option.Set("calculated_price", price)
	amount, _ := price.Get("calculated_amount")
	option.Set("amount", amount)
	inclusive, _ := price.Get("is_calculated_price_tax_inclusive")
	option.Set("is_tax_inclusive", inclusive)
	out := dal.NewRecord()
	out.Set("shipping_option", option)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
