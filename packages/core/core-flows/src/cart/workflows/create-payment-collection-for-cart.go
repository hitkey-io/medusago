package workflows

import (
	"strings"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	paymentsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment-collection/steps"
	paymentworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment-collection/workflows"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

var validateExistingPaymentCollectionStep = composer.CreateStep("validate-existing-payment-collection",
	func(ctx composer.StepExecutionContext, cart *dal.Record) (composer.StepResponse[any], error) {
		if Truthy(Get(cart, "payment_collection")) {
			return composer.StepResponse[any]{}, plainError("Cart " + Str(cart, "id") + " already has a payment collection")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// CreatePaymentCollectionForCartWorkflow mirrors createPaymentCollectionForCartWorkflow.
var CreatePaymentCollectionForCartWorkflow = composer.CreateWorkflow("create-payment-collection-for-cart",
	func(e *composer.Execution, cartID string) (*dal.Record, error) {
		if err := acquireLock(e, cartID, 2, 10); err != nil {
			return nil, err
		}
		cart, err := fetchCart(e, "use-remote-query", cartID, []string{"id", "completed_at", "currency_code", "total", "raw_total", "payment_collection.id"}, true)
		if err != nil {
			return nil, err
		}
		err = composer.Parallelize(
			func() error { _, err := steps.ValidateCartStep.Run(e, cart); return err },
			func() error { _, err := validateExistingPaymentCollectionStep.Run(e, cart); return err },
		)
		if err != nil {
			return nil, err
		}
		data := dal.NewRecord()
		data.Set("currency_code", Get(cart, "currency_code"))
		data.Set("amount", Get(cart, "raw_total"))
		created, err := paymentsteps.CreatePaymentCollectionsStep.Run(e, []*dal.Record{data})
		if err != nil {
			return nil, err
		}
		c := dal.NewRecord()
		c.Set("cart_id", cartID)
		p := dal.NewRecord()
		p.Set("payment_collection_id", Get(created[0], "id"))
		link := dal.NewRecord()
		link.Set(utilsmodulessdk.CART, c)
		link.Set(utilsmodulessdk.PAYMENT, p)
		if _, err := commonsteps.CreateRemoteLinkStep.Config("cart-payment-collection-link", composer.StepOptions{}).Run(e, []*dal.Record{link}); err != nil {
			return nil, err
		}
		if err := releaseLock(e, cartID); err != nil {
			return nil, err
		}
		return created[0], nil
	})

// UpdatePaymentCollectionStep mirrors updatePaymentCollectionStep.
var UpdatePaymentCollectionStep = composer.CreateStep("update-payment-collection",
	func(ctx composer.StepExecutionContext, in *dal.Record) (composer.StepResponse[any], error) {
		sel, update := Rec(in, "selector"), Rec(in, "update")
		if !IsPresent(in) || !IsPresent(sel) {
			return composer.NewStepResponse[any]([]any{}, any([]*dal.Record{})), nil
		}
		svc := paymentsteps.PaymentOf(ctx.Container)
		prev, err := svc.ListBySuffix(ctx.Context, "PaymentCollections", sel, utilsmodulessdk.FindConfig{Select: append([]string{"id"}, update.Keys()...)})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		u := Copy(update)
		if code, ok := Get(u, "currency_code").(string); ok && code != "" {
			u.Set("currency_code", strings.ToLower(code))
		}
		updated, err := commonsteps.UpdateBySelector(ctx.Context, svc, "PaymentCollection", sel, u)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		// updatePaymentCollections(selector, data) serializes result[0]
		// (data is not an array)
		if len(updated) == 0 {
			return composer.NewStepResponse[any](nil, any(prev)), nil
		}
		return composer.NewStepResponse[any](updated[0], any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		_, err := paymentsteps.PaymentOf(ctx.Container).Internal("PaymentCollection").Upsert(ctx.Context, prev)
		return err
	})

func refreshPaymentCollection(e *composer.Execution, cart *dal.Record) error {
	pc := Rec(cart, "payment_collection")
	var ids []any
	for _, s := range Records(Get(pc, "payment_sessions")) {
		ids = append(ids, Get(s, "id"))
	}
	if ids == nil {
		ids = []any{}
	}
	var update *dal.Record
	if IsPresent(Get(pc, "id")) {
		sel := dal.NewRecord()
		sel.Set("id", Get(pc, "id"))
		u := dal.NewRecord()
		u.Set("amount", Get(cart, "raw_total"))
		u.Set("currency_code", Get(cart, "currency_code"))
		update = dal.NewRecord()
		update.Set("selector", sel)
		update.Set("update", u)
	}
	return composer.Parallelize(
		func() error { _, err := paymentworkflows.DeletePaymentSessionsWorkflow.RunAsStep(e, ids); return err },
		func() error { _, err := UpdatePaymentCollectionStep.Run(e, update); return err },
	)
}

// The Go bodies take one property of the upstream input object.
func init() {
	CreatePaymentCollectionForCartWorkflow.InputFrom("cart_id")
}
