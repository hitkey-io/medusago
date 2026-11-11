// Package types mirrors @medusajs/payment-stripe/src/types.
package types

import "github.com/hitkey-io/medusago/packages/core/utils/src/dal"

// StripeOptions mirrors StripeOptions.
type StripeOptions struct {
	APIKey                     *string
	WebhookSecret              *string
	APIVersion                 *string
	Capture                    bool
	AutomaticPaymentMethods    bool
	PaymentMethodConfiguration *string
	PaymentDescription         *string
	OxxoExpiresDays            float64
	AsyncPaymentMethodTypes    []string
	HasAsyncPaymentMethodTypes bool
}

func str(m map[string]any, k string) *string {
	if v, ok := m[k]; ok && v != nil {
		s, ok := v.(string)
		if ok {
			return &s
		}
	}
	return nil
}

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	case int:
		return t != 0
	}
	return true
}

// ParseStripeOptions reads the provider options.
func ParseStripeOptions(m map[string]any) StripeOptions {
	o := StripeOptions{
		APIKey:                     str(m, "apiKey"),
		WebhookSecret:              str(m, "webhookSecret"),
		APIVersion:                 str(m, "apiVersion"),
		Capture:                    truthy(m["capture"]),
		AutomaticPaymentMethods:    truthy(m["automaticPaymentMethods"]),
		PaymentMethodConfiguration: str(m, "paymentMethodConfiguration"),
		PaymentDescription:         str(m, "paymentDescription"),
	}
	switch n := m["oxxoExpiresDays"].(type) {
	case float64:
		o.OxxoExpiresDays = n
	case int:
		o.OxxoExpiresDays = float64(n)
	}
	if l, ok := m["asyncPaymentMethodTypes"]; ok && l != nil {
		o.HasAsyncPaymentMethodTypes = true
		switch t := l.(type) {
		case []string:
			o.AsyncPaymentMethodTypes = t
		case []any:
			for _, x := range t {
				if s, ok := x.(string); ok {
					o.AsyncPaymentMethodTypes = append(o.AsyncPaymentMethodTypes, s)
				}
			}
		}
	}
	return o
}

// PaymentIntentOptions mirrors PaymentIntentOptions (absent fields are nil).
type PaymentIntentOptions struct {
	CaptureMethod        *string
	SetupFutureUsage     *string
	PaymentMethodTypes   []string
	PaymentMethodOptions *dal.Record
}

// ErrorCodes mirrors ErrorCodes.
const (
	PaymentIntentUnexpectedState = "payment_intent_unexpected_state"
	ChargeAlreadyRefunded        = "charge_already_refunded"
)

// ErrorIntentStatus mirrors ErrorIntentStatus.
const (
	IntentSucceeded = "succeeded"
	IntentCanceled  = "canceled"
)

// PaymentProviderKeys mirrors PaymentProviderKeys.
const (
	Stripe     = "stripe"
	Oxxo       = "stripe-oxxo"
	BanContact = "stripe-bancontact"
	Blik       = "stripe-blik"
	Giropay    = "stripe-giropay"
	Ideal      = "stripe-ideal"
	Przelewy24 = "stripe-przelewy24"
	PromptPay  = "stripe-promptpay"
)
