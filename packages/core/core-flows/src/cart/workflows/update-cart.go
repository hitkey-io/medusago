package workflows

import (
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type prepareUpdateInput struct {
	input        *dal.Record
	region       *dal.Record
	customer     steps.CustomerData
	salesChannel *dal.Record
	cartToUpdate *dal.Record
}

var prepareCartToUpdateStep = composer.CreateStep("prepare-cart-to-update",
	func(ctx composer.StepExecutionContext, d prepareUpdateInput) (composer.StepResponse[*dal.Record], error) {
		data := dal.NewRecord()
		for _, k := range d.input.Keys() {
			if k == "promo_codes" || k == "additional_data" {
				continue
			}
			v, _ := d.input.Get(k)
			data.Set(k, v)
		}
		data.Set("currency_code", Get(d.region, "currency_code"))
		data.Set("region_id", Get(d.region, "id"))
		regionIsNew := Get(d.region, "id") != Get(Rec(d.cartToUpdate, "region"), "id")
		sa := Rec(d.input, "shipping_address")
		if code := Str(sa, "country_code"); code != "" {
			var country *dal.Record
			for _, c := range Records(Get(d.region, "countries")) {
				if Str(c, "iso_2") == code {
					country = c
					break
				}
			}
			if country == nil {
				return composer.StepResponse[*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Country with code %s is not within region %v", code, Get(d.region, "name")))
			}
			a := Copy(sa)
			a.Set("country_code", Get(country, "iso_2"))
			data.Set("shipping_address", a)
		}
		if regionIsNew {
			countries := Records(Get(d.region, "countries"))
			if len(countries) == 1 {
				a := dal.NewRecord()
				a.Set("country_code", Get(countries[0], "iso_2"))
				data.Set("shipping_address", a)
			}
			if Str(Rec(data, "shipping_address"), "country_code") == "" {
				data.Set("shipping_address", nil)
			}
		}
		if Has(d.input, "email") && d.customer.Customer != nil {
			data.Set("customer_id", Get(d.customer.Customer, "id"))
			if Truthy(Get(d.customer.Customer, "has_account")) {
				data.Set("email", Get(d.input, "email"))
			} else {
				data.Set("email", d.customer.Email)
			}
		}
		if Has(d.input, "sales_channel_id") {
			data.Set("sales_channel_id", Get(d.salesChannel, "id"))
		}
		return composer.NewStepResponse(data), nil
	}, nil)

type cartsBefore struct {
	carts     []*dal.Record
	addresses []*dal.Record
}

// UpdateCartsStep mirrors updateCartsStep.
var UpdateCartsStep = composer.CreateStep("update-carts",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		svc := steps.CartOf(ctx.Container)
		if len(data) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any(cartsBefore{})), nil
		}
		var ids []any
		for _, d := range data {
			ids = append(ids, Get(d, "id"))
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		before, err := svc.ListBySuffix(ctx.Context, "Carts", f, utilsmodulessdk.FindConfig{Select: []string{"id", "region_id", "customer_id", "sales_channel_id", "email", "currency_code", "metadata", "completed_at"}})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		var addrIDs []any
		var addrUpdates []any
		for _, d := range data {
			for _, k := range []string{"shipping_address", "billing_address"} {
				if a := Rec(d, k); a != nil && Truthy(Get(a, "id")) {
					addrIDs = append(addrIDs, Get(a, "id"))
					addrUpdates = append(addrUpdates, a)
				}
			}
		}
		af := dal.NewRecord()
		if addrIDs == nil {
			addrIDs = []any{}
		}
		af.Set("id", addrIDs)
		addrBefore, err := svc.ListBySuffix(ctx.Context, "Addresses", af, utilsmodulessdk.FindConfig{})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		if len(addrUpdates) > 0 {
			if _, err := svc.Internal("Address").Update(ctx.Context, addrUpdates); err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
		}
		var updates []any
		for _, d := range data {
			u := Copy(d)
			if code, ok := Get(u, "currency_code").(string); ok && code != "" {
				u.Set("currency_code", strings.ToLower(code))
			}
			updates = append(updates, u)
		}
		res, err := svc.Internal("Cart").Update(ctx.Context, updates)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		updated, _ := res.([]*dal.Record)
		return composer.NewStepResponse(updated, any(cartsBefore{before, addrBefore})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		b, ok := data.(cartsBefore)
		if !ok {
			return nil
		}
		svc := steps.CartOf(ctx.Container)
		var addrs []any
		for _, a := range b.addresses {
			addrs = append(addrs, a)
		}
		if len(addrs) > 0 {
			if _, err := svc.Internal("Address").Update(ctx.Context, addrs); err != nil {
				return err
			}
		}
		var carts []any
		for _, c := range b.carts {
			carts = append(carts, c)
		}
		if len(carts) == 0 {
			return nil
		}
		_, err := svc.Internal("Cart").Update(ctx.Context, carts)
		return err
	})

// UpdateCartWorkflow mirrors updateCartWorkflow; the input is the request
// body plus id.
var UpdateCartWorkflow = composer.CreateWorkflow("update-cart",
	func(e *composer.Execution, input *dal.Record) (any, error) {
		id := Str(input, "id")
		if err := acquireLock(e, id, 2, 10); err != nil {
			return nil, err
		}
		cartToUpdate, err := fetchCart(e, "get-cart", id, []string{"id", "email", "customer_id", "sales_channel_id", "locale", "shipping_address.*", "region.*", "region.countries.*"}, true)
		if err != nil {
			return nil, err
		}
		if _, err := steps.ValidateCartStep.Run(e, cartToUpdate); err != nil {
			return nil, err
		}
		salesChannelID := Coalesce(Get(input, "sales_channel_id"), Get(cartToUpdate, "sales_channel_id"))
		email := Coalesce(Get(input, "email"), Get(cartToUpdate, "email"))
		var salesChannel *dal.Record
		var customer steps.CustomerData
		hasAccount := false
		err = composer.Parallelize(
			func() error {
				var err error
				salesChannel, err = steps.FindSalesChannelStep.Run(e, salesChannelID)
				return err
			},
			func() error {
				var err error
				customer, err = steps.FindOrCreateCustomerStep.Run(e, steps.FindOrCreateCustomerInput{CustomerID: Get(cartToUpdate, "customer_id"), Email: email, HasAccount: &hasAccount})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		if _, err := steps.ValidateSalesChannelStep.Run(e, salesChannel); err != nil {
			return nil, err
		}
		var newRegion *dal.Record
		if rid := Str(input, "region_id"); rid != "" {
			isList := false
			f := dal.NewRecord()
			f.Set("id", rid)
			p := dal.NewRecord()
			p.Set("take", float64(1))
			res, err := commonsteps.UseQueryGraphStep.Config("get-region", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
				GraphInput: query.GraphInput{Entity: "region", Fields: []string{"id", "countries.*", "currency_code", "name"}, Filters: f, Pagination: p},
				Options:    joiner.Options{ThrowIfKeyNotFound: true}, IsList: &isList,
			})
			if err != nil {
				return nil, err
			}
			newRegion, _ = res.Data.(*dal.Record)
		}
		taxChanged := false
		if next := Rec(input, "shipping_address"); next != nil {
			prev := Rec(cartToUpdate, "shipping_address")
			changed := func(n, p any) bool { return n != nil && n != p }
			lower := func(v any) any {
				if s, ok := v.(string); ok {
					return strings.ToLower(s)
				}
				return v
			}
			taxChanged = changed(Get(next, "id"), Get(prev, "id")) ||
				changed(lower(Get(next, "country_code")), lower(Get(prev, "country_code"))) ||
				changed(Get(next, "province"), Get(prev, "province")) ||
				changed(Get(next, "postal_code"), Get(prev, "postal_code")) ||
				changed(Get(next, "city"), Get(prev, "city"))
		}
		region := newRegion
		if region == nil {
			region = Rec(cartToUpdate, "region")
		}
		cartInput, err := prepareCartToUpdateStep.Run(e, prepareUpdateInput{input: input, region: region, customer: customer, salesChannel: salesChannel, cartToUpdate: cartToUpdate})
		if err != nil {
			return nil, err
		}
		vh := dal.NewRecord()
		vh.Set("input", cartInput)
		vh.Set("cart", cartToUpdate)
		if err := composer.CreateHook(e, "validate", vh); err != nil {
			return nil, err
		}
		regionUpdated := Has(input, "region_id") && Get(input, "region_id") != Get(Rec(cartToUpdate, "region"), "id")
		var newLocale string
		if l, ok := Get(input, "locale").(string); ok && Has(input, "locale") && l != Str(cartToUpdate, "locale") {
			newLocale = l
		}
		err = composer.Parallelize(
			func() error {
				ev := dal.NewRecord()
				ev.Set("id", id)
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "cart.updated", Data: ev})
				return err
			},
			func() error { _, err := UpdateCartsStep.Run(e, []*dal.Record{cartInput}); return err },
		)
		if err != nil {
			return nil, err
		}
		if regionUpdated {
			f := dal.NewRecord()
			f.Set("cart_id", id)
			f.Set("is_custom_price", true)
			res, err := commonsteps.UseQueryGraphStep.Run(e, commonsteps.UseQueryGraphStepInput{GraphInput: query.GraphInput{Entity: "line_items", Fields: []string{"id"}, Filters: f}})
			if err != nil {
				return nil, err
			}
			var ids []any
			for _, r := range Records(res.Data) {
				ids = append(ids, Get(r, "id"))
			}
			if ids == nil {
				ids = []any{}
			}
			if _, err := deleteLineItemsStep.Run(e, ids); err != nil {
				return nil, err
			}
		}
		refresh := RefreshCartItemsInput{CartID: Str(cartInput, "id"), ForceRefresh: newRegion != nil || newLocale != "" || taxChanged, Locale: newLocale, AdditionalData: Rec(input, "additional_data")}
		if v, ok := input.Get("promo_codes"); ok && v != nil {
			refresh.PromoCodes = asStrings(v)
			if refresh.PromoCodes == nil {
				refresh.PromoCodes = []string{}
			}
			refresh.PromoCodesDefined = true
		}
		cart, err := RefreshCartItemsWorkflow.RunAsStep(e, refresh)
		if err != nil {
			return nil, err
		}
		ch := dal.NewRecord()
		ch.Set("cart", cart)
		ch.Set("additional_data", Get(input, "additional_data"))
		if err := composer.CreateHook(e, "cartUpdated", ch); err != nil {
			return nil, err
		}
		return nil, releaseLock(e, id)
	})
