package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/payment-collection/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func g(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func recs(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		var out []*dal.Record
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

// DeletePaymentSessionsWorkflow mirrors deletePaymentSessionsWorkflow.
var DeletePaymentSessionsWorkflow = composer.CreateWorkflow("delete-payment-sessions",
	func(e *composer.Execution, ids []any) ([]any, error) {
		deleted, err := steps.DeletePaymentSessionsStep.Run(e, ids)
		if err != nil {
			return nil, err
		}
		if len(ids) != len(deleted) {
			return nil, common.NewMedusaError(common.MedusaErrorTypes.UNEXPECTED_STATE, "Could not delete all payment sessions")
		}
		return deleted, nil
	})

// CreatePaymentSessionsInput mirrors CreatePaymentSessionsWorkflowInput.
type CreatePaymentSessionsInput struct {
	PaymentCollectionID string
	ProviderID          string
	CustomerID          string
	Data                *dal.Record
	Context             *dal.Record
}

func remoteOne(e *composer.Execution, name, entry string, id any, fields []string) (*dal.Record, error) {
	v := dal.NewRecord()
	v.Set("id", id)
	list := false
	res, err := commonsteps.UseRemoteQueryStep.Config(name, composer.StepOptions{}).Run(e, commonsteps.RemoteStepInput{EntryPoint: entry, Fields: fields, Variables: v, List: &list})
	if err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, nil
	}
	r, _ := res.Data[0].(*dal.Record)
	return r, nil
}

// customerAccountHolder is the value of the "customer-id-exists" when.
type customerAccountHolder struct {
	PaymentCustomer       *dal.Record `json:"paymentCustomer"`
	AccountHolder         *dal.Record `json:"accountHolder"`
	ExistingAccountHolder *dal.Record `json:"existingAccountHolder,omitempty"`
}

// CreatePaymentSessionsWorkflow mirrors createPaymentSessionsWorkflow.
var CreatePaymentSessionsWorkflow = composer.CreateWorkflow("create-payment-sessions",
	func(e *composer.Execution, in CreatePaymentSessionsInput) (*dal.Record, error) {
		collection, err := remoteOne(e, "get-payment-collection", "payment_collection", in.PaymentCollectionID, []string{"id", "amount", "currency_code", "payment_sessions.*", "payment_sessions.payment.id", "payment_sessions.payment.canceled_at", "payment_sessions.payment.captured_at", "payment_sessions.payment.amount", "payment_sessions.payment.raw_amount", "payment_sessions.payment.refunds.raw_amount"})
		if err != nil {
			return nil, err
		}
		if _, err := steps.ValidatePaymentProviderInRegionStep.Run(e, struct {
			CollectionID string
			ProviderID   string
		}{in.PaymentCollectionID, in.ProviderID}); err != nil {
			return nil, err
		}
		if _, err := steps.ValidatePaymentSessionsDeletableStep.Run(e, recs(g(collection, "payment_sessions"))); err != nil {
			return nil, err
		}
		holder, err := composer.When(e, "customer-id-exists", in.CustomerID != "", func() (*customerAccountHolder, error) {
			customer, err := remoteOne(e, "get-customer", "customer", in.CustomerID, []string{"id", "email", "company_name", "first_name", "last_name", "phone", "addresses.*", "account_holders.*", "metadata"})
			if err != nil {
				return nil, err
			}
			paymentCustomer := dal.NewRecord()
			if customer != nil {
				for _, k := range customer.Keys() {
					v, _ := customer.Get(k)
					paymentCustomer.Set(k, v)
				}
			}
			addresses := recs(g(customer, "addresses"))
			var billing any
			for _, a := range addresses {
				if b, _ := g(a, "is_default_billing").(bool); b {
					billing = a
					break
				}
			}
			if billing == nil && len(addresses) > 0 {
				billing = addresses[0]
			}
			// addresses?.[0] is undefined without addresses: the key is dropped
			if billing != nil {
				paymentCustomer.Set("billing_address", billing)
			}
			var existing *dal.Record
			for _, ah := range recs(g(customer, "account_holders")) {
				if g(ah, "provider_id") == in.ProviderID {
					existing = ah
					break
				}
			}
			ahc := dal.NewRecord()
			ahc.Set("account_holder", existing)
			ahc.Set("customer", paymentCustomer)
			ahi := dal.NewRecord()
			ahi.Set("provider_id", in.ProviderID)
			ahi.Set("context", ahc)
			accountHolder, err := steps.CreatePaymentAccountHolderStep.Config("", composer.StepOptions{NoCompensation: true}).Run(e, ahi)
			if err != nil {
				return nil, err
			}
			return &customerAccountHolder{PaymentCustomer: paymentCustomer, AccountHolder: accountHolder, ExistingAccountHolder: existing}, nil
		})
		if err != nil {
			return nil, err
		}
		var paymentCustomer, accountHolder, existing *dal.Record
		if holder != nil {
			paymentCustomer, accountHolder, existing = holder.PaymentCustomer, holder.AccountHolder, holder.ExistingAccountHolder
		}
		{
			if existing == nil && accountHolder != nil {
				c := dal.NewRecord()
				c.Set("customer_id", g(paymentCustomer, "id"))
				p := dal.NewRecord()
				p.Set("account_holder_id", g(accountHolder, "id"))
				link := dal.NewRecord()
				link.Set(utilsmodulessdk.CUSTOMER, c)
				link.Set(utilsmodulessdk.PAYMENT, p)
				if _, err := commonsteps.CreateRemoteLinkStep.Run(e, []*dal.Record{link}); err != nil {
					return nil, err
				}
			}
		}
		ctx := dal.NewRecord()
		if in.Context != nil {
			for _, k := range in.Context.Keys() {
				v, _ := in.Context.Get(k)
				ctx.Set(k, v)
			}
		}
		if paymentCustomer != nil {
			ctx.Set("customer", paymentCustomer)
		}
		if accountHolder != nil {
			ctx.Set("account_holder", accountHolder)
		}
		sessionIn := dal.NewRecord()
		sessionIn.Set("payment_collection_id", in.PaymentCollectionID)
		sessionIn.Set("provider_id", in.ProviderID)
		sessionIn.Set("data", in.Data)
		sessionIn.Set("context", ctx)
		sessionIn.Set("amount", g(collection, "amount"))
		sessionIn.Set("currency_code", g(collection, "currency_code"))
		var ids []any
		for _, s := range recs(g(collection, "payment_sessions")) {
			ids = append(ids, g(s, "id"))
		}
		if ids == nil {
			ids = []any{}
		}
		if _, err := DeletePaymentSessionsWorkflow.RunAsStep(e, ids); err != nil {
			return nil, err
		}
		return steps.CreatePaymentSessionStep.Run(e, sessionIn)
	})

// UpdatePaymentSessionClientDataInput mirrors UpdatePaymentSessionClientDataWorkflowInput.
type UpdatePaymentSessionClientDataInput struct {
	PaymentCollectionID any
	PaymentSessionID    any
	ClientData          any
}

// UpdatePaymentSessionClientDataWorkflow mirrors updatePaymentSessionClientDataWorkflow.
var UpdatePaymentSessionClientDataWorkflow = composer.CreateWorkflow("update-payment-session-client-data",
	func(e *composer.Execution, in UpdatePaymentSessionClientDataInput) (*dal.Record, error) {
		f := dal.NewRecord()
		f.Set("id", in.PaymentSessionID)
		res, err := commonsteps.UseQueryGraphStep.Config("get-payment-session", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "payment_session", Fields: []string{"id", "payment_collection_id", "status", "amount", "currency_code", "data"}, Filters: f},
		})
		if err != nil {
			return nil, err
		}
		var session *dal.Record
		if list := recs(res.Data); len(list) > 0 {
			session = list[0]
		}
		if _, err := steps.ValidatePaymentSessionUpdateStep.Run(e, steps.ValidatePaymentSessionUpdateInput{PaymentCollectionID: in.PaymentCollectionID, PaymentSessionID: in.PaymentSessionID, PaymentSession: session}); err != nil {
			return nil, err
		}
		data := dal.NewRecord()
		if d, ok := g(session, "data").(*dal.Record); ok && d != nil {
			for _, k := range d.Keys() {
				v, _ := d.Get(k)
				data.Set(k, v)
			}
		}
		data.Set("client", in.ClientData)
		u := dal.NewRecord()
		u.Set("id", g(session, "id"))
		u.Set("amount", g(session, "amount"))
		u.Set("currency_code", g(session, "currency_code"))
		u.Set("data", data)
		return steps.UpdatePaymentSessionStep.Run(e, u)
	})
