// Package subscribers mirrors packages/medusa/src/subscribers.
package subscribers

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type webhookActions interface {
	GetWebhookActionAndData(ctx context.Context, provider string, payload any) (*dal.Record, error)
}

// PaymentWebhookHandler mirrors paymentWebhookhandler: events without a
// session id or with an unhandled action are ignored. The bundled system
// provider reports not_supported, so processPaymentWorkflow never runs.
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
	case "not_supported", "canceled", "failed", "requires_more", "pending_authorization", "pending":
		return nil
	}
	return fmt.Errorf("processPaymentWorkflow is not ported")
}
