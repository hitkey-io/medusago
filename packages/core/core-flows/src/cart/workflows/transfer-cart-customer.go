package workflows

import (
	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// TransferCartCustomerInput mirrors TransferCartCustomerWorkflowInput & AdditionalData.
type TransferCartCustomerInput struct {
	ID             string
	CustomerID     any
	AdditionalData *dal.Record
}

// TransferCartCustomerWorkflow mirrors transferCartCustomerWorkflow: the cart
// is locked while it is transferred.
var TransferCartCustomerWorkflow = composer.CreateWorkflow("transfer-cart-customer",
	func(e *composer.Execution, in TransferCartCustomerInput) (any, error) {
		f := dal.NewRecord()
		f.Set("id", in.ID)
		res, err := commonsteps.UseQueryGraphStep.Config("get-cart", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "cart", Fields: []string{"id", "email", "customer_id", "customer.has_account", "shipping_address.*",
				"region.*", "region.countries.*"}, Filters: f},
			Options: joiner.Options{ThrowIfKeyNotFound: true},
		})
		if err != nil {
			return nil, err
		}
		var cart *dal.Record
		if list := Records(res.Data); len(list) > 0 {
			cart = list[0]
		}
		hookIn := dal.NewRecord()
		hookIn.Set("input", in)
		hookIn.Set("cart", cart)
		if err := composer.CreateHook(e, "validate", hookIn); err != nil {
			return nil, err
		}
		cf := dal.NewRecord()
		cf.Set("id", in.CustomerID)
		isList := false
		cres, err := commonsteps.UseQueryGraphStep.Config("get-customer", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "customer", Fields: []string{"id", "email"}, Filters: cf},
			Options:    joiner.Options{ThrowIfKeyNotFound: true},
			IsList:     &isList,
		})
		if err != nil {
			return nil, err
		}
		customer, _ := cres.Data.(*dal.Record)
		if Get(Rec(cart, "customer"), "id") == Get(customer, "id") {
			return nil, nil
		}
		if err := acquireLock(e, Str(cart, "id"), 2, 10); err != nil {
			return nil, err
		}
		u := dal.NewRecord()
		u.Set("id", Get(cart, "id"))
		u.Set("customer_id", Get(customer, "id"))
		u.Set("email", Get(customer, "email"))
		if _, err := UpdateCartsStep.Run(e, []*dal.Record{u}); err != nil {
			return nil, err
		}
		if _, err := RefreshCartItemsWorkflow.RunAsStep(e, RefreshCartItemsInput{CartID: in.ID, ForceRefresh: true, AdditionalData: in.AdditionalData}); err != nil {
			return nil, err
		}
		// customer.customer_id is undefined upstream (the event carries no customer)
		ev := dal.NewRecord()
		ev.Set("id", in.ID)
		err = composer.Parallelize(
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "cart.customer_transferred", Data: ev})
				return err
			},
			func() error { return releaseLock(e, Str(cart, "id")) },
		)
		return nil, err
	})
