// Package subscribers mirrors packages/medusa/src/subscribers.
package subscribers

import (
	"context"
	"fmt"

	paymentworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment/workflows"
	paymenttypes "github.com/hitkey-io/medusago/packages/core/types/src/payment"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type webhookActions interface {
	GetWebhookActionAndData(ctx context.Context, provider string, payload any) (*dal.Record, error)
}

// PaymentWebhookHandler mirrors paymentWebhookhandler: events without a
// session id or with an action processPaymentWorkflow does not handle are
// ignored; the others run processPaymentWorkflow through the workflow
// engine.
func PaymentWebhookHandler(ctx context.Context, c composer.Container, data any) error {
	svc, _ := c.Resolve(utilsmodulessdk.PAYMENT).(webhookActions)
	input, _ := data.(*dal.Record)
	if svc == nil || input == nil {
		return nil
	}
	provider, _ := input.Get("provider")
	payload, _ := input.Get("payload")
	processed, err := svc.GetWebhookActionAndData(ctx, fmt.Sprint(provider), payload)
	if err != nil {
		return err
	}
	d, _ := processed.Get("data")
	pd, _ := d.(*dal.Record)
	if pd == nil {
		return nil
	}
	if sid, _ := pd.Get("session_id"); sid == nil || sid == "" {
		return nil
	}
	switch action, _ := processed.Get("action"); action {
	case paymenttypes.ActionNotSupported, paymenttypes.ActionCanceled, paymenttypes.ActionFailed, paymenttypes.ActionRequiresMore,
		paymenttypes.ActionPendingAuthorization, paymenttypes.ActionPending:
		return nil
	}
	_, _, err = composer.RunByID(ctx, c, paymentworkflows.ProcessPaymentWorkflowID, processed, composer.RunOptions{Notify: true})
	return err
}
