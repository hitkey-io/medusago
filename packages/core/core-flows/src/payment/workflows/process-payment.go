package workflows

import (
	"errors"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	paymentsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	paymenttypes "github.com/hitkey-io/medusago/packages/core/types/src/payment"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ProcessPaymentWorkflowID mirrors processPaymentWorkflowId.
const ProcessPaymentWorkflowID = "process-payment-workflow"

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case bool:
		return t
	case float64:
		return t != 0
	}
	return true
}

// ProcessPaymentWorkflow mirrors processPaymentWorkflow: a webhook action
// (WebhookActionResult) captures the payment, authorizes the session or
// completes the cart. The lock steps are not run (the locking module is not
// ported).
var ProcessPaymentWorkflow = composer.CreateWorkflow(ProcessPaymentWorkflowID,
	func(e *composer.Execution, input *dal.Record) (any, error) {
		data := asRec(g(input, "data"))
		sessionID := g(data, "session_id")
		amount := g(data, "amount")
		action := g(input, "action")

		graph := func(name, entity string, fields []string, filters *dal.Record, isList bool) (commonsteps.QueryGraphResult, error) {
			in := commonsteps.UseQueryGraphStepInput{GraphInput: query.GraphInput{Entity: entity, Fields: fields, Filters: filters}}
			if !isList {
				f := false
				in.IsList = &f
			}
			return commonsteps.UseQueryGraphStep.Config(name, composer.StepOptions{}).Run(e, in)
		}

		paymentRes, err := graph("payment-query", "payment", []string{"id"}, kv("payment_session_id", sessionID), true)
		if err != nil {
			return nil, err
		}
		paymentData := recs(paymentRes.Data)
		sessionRes, err := graph("payment-session-query", "payment_session", []string{"payment_collection_id"}, kv("id", sessionID), true)
		if err != nil {
			return nil, err
		}
		sessions := recs(sessionRes.Data)
		if len(sessions) == 0 {
			return nil, errors.New("Cannot read properties of undefined (reading 'payment_collection_id')")
		}
		cartPCRes, err := graph("cart-payment-query", "cart_payment_collection", []string{"cart_id"}, kv("payment_collection_id", g(sessions[0], "payment_collection_id")), true)
		if err != nil {
			return nil, err
		}
		cartPaymentCollection := recs(cartPCRes.Data)
		// Without a session id the queries resolve to arbitrary rows: no cart.
		var cartID any
		if truthy(sessionID) && len(cartPaymentCollection) > 0 {
			cartID = g(cartPaymentCollection[0], "cart_id")
		}
		orderRes, err := graph("cart-order-query", "order_cart", []string{"id"}, kv("cart_id", cartID), false)
		if err != nil {
			return nil, err
		}
		order := asRec(orderRes.Data)

		hasSession := truthy(sessionID)
		sid, _ := sessionID.(string)

		if hasSession && action == paymenttypes.ActionSuccessful && len(paymentData) > 0 {
			c := dal.NewRecord()
			c.Set("payment_id", g(paymentData[0], "id"))
			c.Set("amount", amount)
			if _, err := CapturePaymentWorkflow.RunAsStep(e, c); err != nil {
				return nil, err
			}
		}

		if hasSession && action == paymenttypes.ActionSuccessful && len(paymentData) == 0 {
			autocapturePayment, err := paymentsteps.AuthorizePaymentSessionStep.Config("authorize-payment-session-autocapture", composer.StepOptions{}).
				Run(e, paymentsteps.AuthorizePaymentSessionStepInput{ID: sid, Context: dal.NewRecord()})
			if err != nil {
				return nil, err
			}
			// A deferred authorization returns no payment to capture yet.
			if autocapturePayment != nil {
				c := dal.NewRecord()
				c.Set("payment_id", g(autocapturePayment, "id"))
				c.Set("amount", amount)
				if _, err := CapturePaymentWorkflow.RunAsStep(e, c); err != nil {
					return nil, err
				}
			}
		}

		// Without a cart, the session is authorized here; with one, the
		// complete cart workflow authorizes it.
		if len(cartPaymentCollection) == 0 && action == paymenttypes.ActionAuthorized && hasSession {
			if _, err := paymentsteps.AuthorizePaymentSessionStep.Config("authorize-payment-session", composer.StepOptions{}).
				Run(e, paymentsteps.AuthorizePaymentSessionStepInput{ID: sid, Context: dal.NewRecord()}); err != nil {
				return nil, err
			}
		}

		// An order created with a pending authorization gets its payment.
		if order != nil && len(paymentData) == 0 && len(cartPaymentCollection) > 0 && action == paymenttypes.ActionAuthorized && hasSession {
			if _, err := paymentsteps.AuthorizePaymentSessionStep.Config("authorize-payment-session-deferred", composer.StepOptions{}).
				Run(e, paymentsteps.AuthorizePaymentSessionStepInput{ID: sid, Context: dal.NewRecord()}); err != nil {
				return nil, err
			}
		}

		if hasSession && len(cartPaymentCollection) > 0 && order == nil {
			cid, _ := g(cartPaymentCollection[0], "cart_id").(string)
			if _, err := paymentsteps.CompleteCartAfterPaymentStep.Config("", composer.StepOptions{ContinueOnPermanentFailure: true}).
				Run(e, paymentsteps.CompleteCartAfterPaymentStepInput{CartID: cid}); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
