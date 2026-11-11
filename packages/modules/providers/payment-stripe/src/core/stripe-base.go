// Package core mirrors @medusajs/payment-stripe/src/core.
package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	paymenttypes "github.com/hitkey-io/medusago/packages/core/types/src/payment"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/types"
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/utils"
)

const (
	defaultStripeAPIVersion             = "2024-04-10"
	allowedPaymentMethodTypesAPIVersion = "2026-07-29"
)

var (
	warnOnce sync.Once
	// RetryBaseDelay is the base delay of executeWithRetry (1s upstream).
	RetryBaseDelay = time.Second
)

// ValidateOptions mirrors StripeBase.validateOptions.
func ValidateOptions(options map[string]any) error {
	if _, ok := options["apiKey"]; !ok {
		return errors.New("Required option `apiKey` is missing in Stripe plugin")
	}
	if _, ok := options["webhookSecret"]; !ok {
		warnOnce.Do(func() {
			slog.Warn("Option `webhookSecret` is missing in Stripe plugin. Webhook signature verification will fail, so webhook-dependent payment flows (e.g. 3D Secure, Klarna/Affirm redirects, async capture) will not be able to update orders, leaving them pending after successful charges. Set `webhookSecret` if you rely on Stripe webhooks.")
		})
	}
	return nil
}

// StripeBase mirrors StripeBase: PaymentIntentOptions is the variant's
// paymentIntentOptions getter.
type StripeBase struct {
	Options              types.StripeOptions
	PaymentIntentOptions func(o types.StripeOptions) types.PaymentIntentOptions
	stripe               *Client
}

var _ paymenttypes.IPaymentProvider = (*StripeBase)(nil)

// NewStripeBase mirrors the StripeBase constructor.
func NewStripeBase(options map[string]any, intentOptions func(o types.StripeOptions) types.PaymentIntentOptions) *StripeBase {
	o := types.ParseStripeOptions(options)
	apiKey := ""
	if o.APIKey != nil {
		apiKey = *o.APIKey
	}
	return &StripeBase{Options: o, PaymentIntentOptions: intentOptions, stripe: NewClient(apiKey, apiVersion(o))}
}

func apiVersion(o types.StripeOptions) string {
	if o.APIVersion != nil {
		return *o.APIVersion
	}
	return defaultStripeAPIVersion
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func rec(v any) *dal.Record {
	r, _ := v.(*dal.Record)
	return r
}

func kv(pairs ...any) *dal.Record {
	r := dal.NewRecord()
	for i := 0; i+1 < len(pairs); i += 2 {
		r.Set(pairs[i].(string), pairs[i+1])
	}
	return r
}

// setIf sets the key unless the value is undefined (nil here stands for
// undefined: these values never hold null).
func setIf(r *dal.Record, k string, v any) {
	if v != nil {
		r.Set(k, v)
	}
}

// copyKey copies a key when the source defines it (null included), like
// `res.k = extra?.k` whose undefined values are not sent.
func copyKey(dst *dal.Record, dk string, src *dal.Record, sk string) {
	if src == nil {
		return
	}
	if v, ok := src.Get(sk); ok {
		dst.Set(dk, v)
	}
}

func idempotencyKey(c *dal.Record) string {
	s, _ := get(c, "idempotency_key").(string)
	return s
}

func stringsToAny(l []string) []any {
	out := make([]any, len(l))
	for i, s := range l {
		out[i] = s
	}
	return out
}

func anyToStrings(v any) ([]string, bool) {
	switch t := v.(type) {
	case []string:
		return t, true
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			s, _ := x.(string)
			out = append(out, s)
		}
		return out, true
	}
	return nil, false
}

// getPaymentMethodTypesParams mirrors getPaymentMethodTypesParams.
func (s *StripeBase) getPaymentMethodTypesParams(r *dal.Record, types_ []string, has bool) {
	if !has {
		return
	}
	if apiVersion(s.Options) >= allowedPaymentMethodTypesAPIVersion {
		r.Set("allowed_payment_method_types", stringsToAny(types_))
	} else {
		r.Set("payment_method_types", stringsToAny(types_))
	}
}

// NormalizePaymentIntentParameters mirrors normalizePaymentIntentParameters.
func (s *StripeBase) NormalizePaymentIntentParameters(extra *dal.Record) *dal.Record {
	res := dal.NewRecord()
	intent := s.PaymentIntentOptions(s.Options)
	pmTypes, hasTypes := anyToStrings(get(extra, "allowed_payment_method_types"))
	if !hasTypes {
		pmTypes, hasTypes = anyToStrings(get(extra, "payment_method_types"))
	}
	if !hasTypes && intent.PaymentMethodTypes != nil {
		pmTypes, hasTypes = intent.PaymentMethodTypes, true
	}

	description := get(extra, "payment_description")
	if description == nil && s.Options.PaymentDescription != nil {
		description = *s.Options.PaymentDescription
	}
	setIf(res, "description", description)

	captureMethod := get(extra, "capture_method")
	if captureMethod == nil && intent.CaptureMethod != nil {
		captureMethod = *intent.CaptureMethod
	}
	if captureMethod == nil {
		if s.Options.Capture {
			captureMethod = "automatic"
		} else {
			captureMethod = "manual"
		}
	}
	res.Set("capture_method", captureMethod)

	setupFutureUsage := get(extra, "setup_future_usage")
	if setupFutureUsage == nil && intent.SetupFutureUsage != nil {
		setupFutureUsage = *intent.SetupFutureUsage
	}
	setIf(res, "setup_future_usage", setupFutureUsage)

	s.getPaymentMethodTypesParams(res, pmTypes, hasTypes)

	copyKey(res, "payment_method_data", extra, "payment_method_data")

	pmOptions := get(extra, "payment_method_options")
	if pmOptions == nil && intent.PaymentMethodOptions != nil {
		pmOptions = intent.PaymentMethodOptions
	}
	setIf(res, "payment_method_options", pmOptions)

	automatic := get(extra, "automatic_payment_methods")
	if automatic == nil && s.Options.AutomaticPaymentMethods {
		automatic = kv("enabled", true)
	}
	setIf(res, "automatic_payment_methods", automatic)

	copyKey(res, "off_session", extra, "off_session")
	copyKey(res, "confirm", extra, "confirm")
	copyKey(res, "payment_method", extra, "payment_method")
	copyKey(res, "return_url", extra, "return_url")
	copyKey(res, "shared_payment_token", extra, "shared_payment_token")

	if len(pmTypes) == 0 {
		pmc := get(extra, "payment_method_configuration")
		if pmc == nil && s.Options.PaymentMethodConfiguration != nil {
			pmc = *s.Options.PaymentMethodConfiguration
		}
		setIf(res, "payment_method_configuration", pmc)
	}
	return res
}

// handled mirrors HandledErrorType.
type handled struct {
	retry bool
	data  *dal.Record
}

// handleStripeError mirrors handleStripeError.
func (s *StripeBase) handleStripeError(err error) (handled, error) {
	var se *StripeError
	if !errors.As(err, &se) {
		return handled{}, s.buildError("An error occurred in InitiatePayment during creation of stripe payment intent", err)
	}
	switch se.Type {
	case "StripeCardError":
		if pi := se.PaymentIntent(); pi != nil {
			return handled{data: pi}, nil
		}
		return handled{}, s.buildError("An error occurred in InitiatePayment during creation of stripe payment intent", err)
	case "StripeConnectionError", "StripeRateLimitError":
		return handled{retry: true}, nil
	case "StripeAPIError":
		return handled{data: kv("indeterminate_due_to", "stripe_api_error")}, nil
	}
	return handled{}, s.buildError("An error occurred in InitiatePayment during creation of stripe payment intent", err)
}

// executeWithRetry mirrors executeWithRetry (3 retries, 1s base delay).
func (s *StripeBase) executeWithRetry(apiCall func() (*dal.Record, error)) (*dal.Record, error) {
	const maxRetries = 3
	for attempt := 1; ; attempt++ {
		out, err := apiCall()
		if err == nil {
			return out, nil
		}
		h, herr := s.handleStripeError(err)
		if herr != nil {
			return nil, herr
		}
		if !h.retry {
			return h.data, nil
		}
		if attempt > maxRetries {
			return nil, s.buildError("An error occurred in InitiatePayment during creation of stripe payment intent", err)
		}
		delay := float64(RetryBaseDelay) * float64(int(1)<<(attempt-1)) * (0.5 + rand.Float64()*0.5)
		time.Sleep(time.Duration(delay))
	}
}

// GetPaymentStatus mirrors getPaymentStatus.
func (s *StripeBase) GetPaymentStatus(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	id, _ := get(rec(get(input, "data")), "id").(string)
	if id == "" {
		return nil, s.buildError("No payment intent ID provided while getting payment status", errors.New("No payment intent ID provided"))
	}
	intent, err := s.stripe.Request(ctx, http.MethodGet, "payment_intents/"+id, kv("expand", []any{"payment_method"}), RequestOptions{})
	if err != nil {
		return nil, err
	}
	return s.getStatus(intent), nil
}

// InitiatePayment mirrors initiatePayment.
func (s *StripeBase) InitiatePayment(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	data := rec(get(input, "data"))
	c := rec(get(input, "context"))
	currency, _ := get(input, "currency_code").(string)
	additional := s.NormalizePaymentIntentParameters(data)

	intentRequest := dal.NewRecord()
	intentRequest.Set("amount", float64(utils.GetSmallestUnit(get(input, "amount"), currency)))
	intentRequest.Set("currency", currency)
	metadata := dal.NewRecord()
	if m := rec(get(data, "metadata")); m != nil {
		for _, k := range m.Keys() {
			v, _ := m.Get(k)
			metadata.Set(k, v)
		}
	}
	metadata.Set("session_id", get(data, "session_id"))
	intentRequest.Set("metadata", metadata)
	for _, k := range additional.Keys() {
		v, _ := additional.Get(k)
		intentRequest.Set(k, v)
	}
	intentRequest.Set("expand", []any{"payment_method"})
	setIf(intentRequest, "customer", get(rec(get(rec(get(c, "account_holder")), "data")), "id"))

	sessionData, err := s.executeWithRetry(func() (*dal.Record, error) {
		return s.stripe.Request(ctx, http.MethodPost, "payment_intents", intentRequest, RequestOptions{IdempotencyKey: idempotencyKey(c)})
	})
	if err != nil {
		return nil, err
	}
	out := dal.NewRecord()
	if sessionData.Has("id") {
		out.Set("id", get(sessionData, "id"))
	} else {
		out.Set("id", get(data, "session_id"))
	}
	st := s.getStatus(sessionData)
	out.Set("status", get(st, "status"))
	out.Set("data", get(st, "data"))
	return out, nil
}

// AuthorizePayment mirrors authorizePayment.
func (s *StripeBase) AuthorizePayment(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	return s.GetPaymentStatus(ctx, input)
}

// CancelPayment mirrors cancelPayment.
func (s *StripeBase) CancelPayment(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	return s.cancelPaymentIntent(ctx, input, []string{types.IntentCanceled})
}

func (s *StripeBase) cancelPaymentIntent(ctx context.Context, input *dal.Record, ignoredStatuses []string) (*dal.Record, error) {
	data := get(input, "data")
	id, _ := get(rec(data), "id").(string)
	if id == "" {
		return kv("data", data), nil
	}
	res, err := s.stripe.Request(ctx, http.MethodPost, "payment_intents/"+id+"/cancel", nil, RequestOptions{IdempotencyKey: idempotencyKey(rec(get(input, "context")))})
	if err != nil {
		var se *StripeError
		if errors.As(err, &se) {
			if pi := se.PaymentIntent(); pi != nil {
				st, _ := pi.Get("status")
				for _, ignored := range ignoredStatuses {
					if st == ignored {
						return kv("data", pi), nil
					}
				}
			}
		}
		return nil, s.buildError("An error occurred in cancelPayment", err)
	}
	return kv("data", res), nil
}

// CapturePayment mirrors capturePayment.
func (s *StripeBase) CapturePayment(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	id, _ := get(rec(get(input, "data")), "id").(string)
	intent, err := s.stripe.Request(ctx, http.MethodPost, "payment_intents/"+id+"/capture", nil, RequestOptions{IdempotencyKey: idempotencyKey(rec(get(input, "context")))})
	if err != nil {
		var se *StripeError
		if errors.As(err, &se) && se.Code() == types.PaymentIntentUnexpectedState {
			if pi := se.PaymentIntent(); pi != nil && get(pi, "status") == types.IntentSucceeded {
				return kv("data", pi), nil
			}
		}
		return nil, s.buildError("An error occurred in capturePayment", err)
	}
	return kv("data", intent), nil
}

// DeletePayment mirrors deletePayment: a succeeded intent can't be
// canceled, but its session can still be deleted.
func (s *StripeBase) DeletePayment(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	return s.cancelPaymentIntent(ctx, input, []string{types.IntentCanceled, types.IntentSucceeded})
}

// RefundPayment mirrors refundPayment.
func (s *StripeBase) RefundPayment(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	data := rec(get(input, "data"))
	id, _ := get(data, "id").(string)
	if id == "" {
		return nil, s.buildError("No payment intent ID provided while refunding payment", errors.New("No payment intent ID provided"))
	}
	currency, _ := get(data, "currency").(string)
	_, err := s.stripe.Request(ctx, http.MethodPost, "refunds", kv(
		"amount", float64(utils.GetSmallestUnit(get(input, "amount"), currency)),
		"payment_intent", id,
	), RequestOptions{IdempotencyKey: idempotencyKey(rec(get(input, "context")))})
	if err != nil {
		var se *StripeError
		if !errors.As(err, &se) || se.Code() != types.ChargeAlreadyRefunded {
			return nil, s.buildError("An error occurred in refundPayment", err)
		}
	}
	return kv("data", get(input, "data")), nil
}

// RetrievePayment mirrors retrievePayment.
func (s *StripeBase) RetrievePayment(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	id, _ := get(rec(get(input, "data")), "id").(string)
	intent, err := s.stripe.Request(ctx, http.MethodGet, "payment_intents/"+id, nil, RequestOptions{})
	if err != nil {
		return nil, s.buildError("An error occurred in retrievePayment", err)
	}
	currency, _ := get(intent, "currency").(string)
	intent.Set("amount", utils.GetAmountFromSmallestUnit(get(intent, "amount"), currency))
	return kv("data", intent), nil
}

// UpdatePayment mirrors updatePayment.
func (s *StripeBase) UpdatePayment(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	data := rec(get(input, "data"))
	currency, _ := get(input, "currency_code").(string)
	amount := get(input, "amount")
	amountNumeric := utils.GetSmallestUnit(amount, currency)
	if amount != nil {
		if a, ok := get(data, "amount").(float64); ok && a == float64(amountNumeric) {
			return s.getStatus(data), nil
		}
	}
	id, _ := get(data, "id").(string)
	sessionData, err := s.stripe.Request(ctx, http.MethodPost, "payment_intents/"+id, kv(
		"amount", float64(amountNumeric),
		"expand", []any{"payment_method"},
	), RequestOptions{IdempotencyKey: idempotencyKey(rec(get(input, "context")))})
	if err != nil {
		return nil, s.buildError("An error occurred in updatePayment", err)
	}
	return s.getStatus(sessionData), nil
}

func billingShipping(customer *dal.Record) *dal.Record {
	ba := rec(get(customer, "billing_address"))
	if ba == nil {
		return nil
	}
	return kv(
		"city", get(ba, "city"),
		"country", get(ba, "country_code"),
		"line1", get(ba, "address_1"),
		"line2", get(ba, "address_2"),
		"postal_code", get(ba, "postal_code"),
		"state", get(ba, "province"),
	)
}

// customerParams mirrors the customers.create/update params.
func customerParams(customer *dal.Record) *dal.Record {
	p := dal.NewRecord()
	copyKey(p, "email", customer, "email")
	name := ""
	if cn, _ := get(customer, "company_name").(string); cn != "" {
		name = cn
	} else {
		first, _ := get(customer, "first_name").(string)
		last, _ := get(customer, "last_name").(string)
		name = strings.TrimSpace(first + " " + last)
	}
	if name != "" {
		p.Set("name", name)
	}
	copyKey(p, "phone", customer, "phone")
	if addr := billingShipping(customer); addr != nil {
		p.Set("address", addr)
	}
	return p
}

// CreateAccountHolder mirrors createAccountHolder.
func (s *StripeBase) CreateAccountHolder(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	c := rec(get(input, "context"))
	if id := get(rec(get(rec(get(c, "account_holder")), "data")), "id"); id != nil && id != "" {
		return kv("id", id), nil
	}
	customer := rec(get(c, "customer"))
	if customer == nil {
		return nil, s.buildError("No customer in context", errors.New("No customer provided while creating account holder"))
	}
	sc, err := s.stripe.Request(ctx, http.MethodPost, "customers", customerParams(customer), RequestOptions{IdempotencyKey: idempotencyKey(c)})
	if err != nil {
		return nil, s.buildError("An error occurred in createAccountHolder when creating a Stripe customer", err)
	}
	return kv("id", get(sc, "id"), "data", sc), nil
}

// UpdateAccountHolder mirrors updateAccountHolder.
func (s *StripeBase) UpdateAccountHolder(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	c := rec(get(input, "context"))
	id, _ := get(rec(get(rec(get(c, "account_holder")), "data")), "id").(string)
	if id == "" {
		return nil, s.buildError("No account holder in context", errors.New("No account holder provided while updating account holder"))
	}
	customer := rec(get(c, "customer"))
	if customer == nil {
		return dal.NewRecord(), nil
	}
	sc, err := s.stripe.Request(ctx, http.MethodPost, "customers/"+id, customerParams(customer), RequestOptions{IdempotencyKey: idempotencyKey(c)})
	if err != nil {
		return nil, s.buildError("An error occurred in updateAccountHolder when updating a Stripe customer", err)
	}
	return kv("data", sc), nil
}

// DeleteAccountHolder mirrors deleteAccountHolder.
func (s *StripeBase) DeleteAccountHolder(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	c := rec(get(input, "context"))
	id, _ := get(rec(get(rec(get(c, "account_holder")), "data")), "id").(string)
	if id == "" {
		return nil, s.buildError("No account holder in context", errors.New("No account holder provided while deleting account holder"))
	}
	if _, err := s.stripe.Request(ctx, http.MethodDelete, "customers/"+id, nil, RequestOptions{}); err != nil {
		return nil, s.buildError("An error occurred in deleteAccountHolder", err)
	}
	return dal.NewRecord(), nil
}

// ListPaymentMethods mirrors listPaymentMethods.
func (s *StripeBase) ListPaymentMethods(ctx context.Context, input *dal.Record) ([]*dal.Record, error) {
	c := rec(get(input, "context"))
	id, _ := get(rec(get(rec(get(c, "account_holder")), "data")), "id").(string)
	if id == "" {
		return []*dal.Record{}, nil
	}
	res, err := s.stripe.Request(ctx, http.MethodGet, "customers/"+id+"/payment_methods", kv("limit", float64(100)), RequestOptions{})
	if err != nil {
		return nil, err
	}
	out := []*dal.Record{}
	if l, ok := get(res, "data").([]any); ok {
		for _, m := range l {
			r := rec(m)
			out = append(out, kv("id", get(r, "id"), "data", r))
		}
	}
	return out, nil
}

// SavePaymentMethod mirrors savePaymentMethod.
func (s *StripeBase) SavePaymentMethod(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	c := rec(get(input, "context"))
	id, _ := get(rec(get(rec(get(c, "account_holder")), "data")), "id").(string)
	if id == "" {
		return nil, s.buildError("Account holder not set while saving a payment method", errors.New("Missing account holder"))
	}
	data := rec(get(input, "data"))
	params := kv("customer", id)
	var pmTypes any
	if data != nil {
		for _, k := range data.Keys() {
			v, _ := data.Get(k)
			if k == "payment_method_types" {
				pmTypes = v
				continue
			}
			params.Set(k, v)
		}
	}
	t, has := anyToStrings(pmTypes)
	s.getPaymentMethodTypesParams(params, t, has)
	resp, err := s.stripe.Request(ctx, http.MethodPost, "setup_intents", params, RequestOptions{IdempotencyKey: idempotencyKey(c)})
	if err != nil {
		return nil, err
	}
	return kv("id", get(resp, "id"), "data", resp), nil
}

// DeletePaymentMethod mirrors deletePaymentMethod.
func (s *StripeBase) DeletePaymentMethod(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	id, _ := get(rec(get(input, "data")), "id").(string)
	if id == "" {
		return nil, s.buildError("Payment method ID not set while deleting a payment method", errors.New("Missing payment method ID"))
	}
	if _, err := s.stripe.Request(ctx, http.MethodPost, "payment_methods/"+id+"/detach", nil, RequestOptions{IdempotencyKey: idempotencyKey(rec(get(input, "context")))}); err != nil {
		return nil, err
	}
	return dal.NewRecord(), nil
}

// getStatus mirrors getStatus.
func (s *StripeBase) getStatus(intent *dal.Record) *dal.Record {
	status := paymenttypes.StatusPending
	switch get(intent, "status") {
	case "requires_payment_method":
		if v := get(intent, "last_payment_error"); v != nil {
			status = paymenttypes.StatusError
		}
	case "requires_confirmation":
	case "processing":
		if s.isAsyncPaymentMethod(rec(get(intent, "payment_method"))) {
			status = paymenttypes.StatusPendingAuthorization
		}
	case "requires_action":
		status = paymenttypes.StatusRequiresMore
	case "canceled":
		status = paymenttypes.StatusCanceled
	case "requires_capture":
		status = paymenttypes.StatusAuthorized
	case "succeeded":
		status = paymenttypes.StatusCaptured
	}
	return kv("status", status, "data", intent)
}

func (s *StripeBase) isAsyncPaymentMethod(pm *dal.Record) bool {
	t := get(pm, "type")
	for _, a := range s.Options.AsyncPaymentMethodTypes {
		if a == t {
			return true
		}
	}
	return false
}

// GetWebhookActionAndData mirrors getWebhookActionAndData.
func (s *StripeBase) GetWebhookActionAndData(ctx context.Context, payload *dal.Record) (*dal.Record, error) {
	event, err := s.ConstructWebhookEvent(payload)
	if err != nil {
		return nil, err
	}
	intent := rec(get(rec(get(event, "data")), "object"))
	currency, _ := get(intent, "currency").(string)
	sessionID := get(rec(get(intent, "metadata")), "session_id")
	if sid, ok := sessionID.(string); !ok || sid == "" {
		return kv("action", paymenttypes.ActionNotSupported), nil
	}
	result := func(action string, amount any) (*dal.Record, error) {
		return kv("action", action, "data", kv("session_id", sessionID, "amount", utils.GetAmountFromSmallestUnit(amount, currency))), nil
	}
	switch get(event, "type") {
	case "payment_intent.created", "payment_intent.processing":
		var pm *dal.Record
		switch t := get(intent, "payment_method").(type) {
		case *dal.Record:
			pm = t
		case string:
			pm, err = s.stripe.Request(ctx, http.MethodGet, "payment_methods/"+t, nil, RequestOptions{})
			if err != nil {
				return nil, err
			}
		}
		action := paymenttypes.ActionPending
		if pm != nil && s.isAsyncPaymentMethod(pm) {
			action = paymenttypes.ActionPendingAuthorization
		}
		return result(action, get(intent, "amount"))
	case "payment_intent.canceled":
		return result(paymenttypes.ActionCanceled, get(intent, "amount"))
	case "payment_intent.payment_failed":
		return result(paymenttypes.ActionFailed, get(intent, "amount"))
	case "payment_intent.requires_action":
		return result(paymenttypes.ActionRequiresMore, get(intent, "amount"))
	case "payment_intent.amount_capturable_updated":
		return result(paymenttypes.ActionAuthorized, get(intent, "amount_capturable"))
	case "payment_intent.partially_funded":
		amount := get(rec(get(rec(get(intent, "next_action")), "display_bank_transfer_instructions")), "amount_remaining")
		if amount == nil {
			amount = get(intent, "amount")
		}
		return result(paymenttypes.ActionRequiresMore, amount)
	case "payment_intent.succeeded":
		return result(paymenttypes.ActionSuccessful, get(intent, "amount_received"))
	}
	return kv("action", paymenttypes.ActionNotSupported), nil
}

// ConstructWebhookEvent mirrors constructWebhookEvent.
func (s *StripeBase) ConstructWebhookEvent(payload *dal.Record) (*dal.Record, error) {
	signature, _ := get(rec(get(payload, "headers")), "stripe-signature").(string)
	var raw []byte
	switch t := get(payload, "rawData").(type) {
	case []byte:
		raw = t
	case string:
		raw = []byte(t)
	}
	secret := ""
	if s.Options.WebhookSecret != nil {
		secret = *s.Options.WebhookSecret
	}
	event, err := ConstructEvent(raw, signature, secret)
	if err != nil {
		return nil, s.buildError("An error occurred in constructWebhookEvent", err)
	}
	return event, nil
}

// buildError mirrors buildError: `${message}: ${error.message}. ${detail}`.
func (s *StripeBase) buildError(message string, err error) error {
	detail := ""
	var se *StripeError
	if errors.As(err, &se) && se.Raw != nil {
		if d, ok := se.Raw.Get("detail"); ok {
			detail = jsString(d)
		}
	}
	return errors.New(strings.TrimSpace(fmt.Sprintf("%s: %s. %s", message, err.Error(), detail)))
}

func jsString(v any) string {
	switch t := v.(type) {
	case nil:
		return "undefined"
	case string:
		return t
	case float64:
		return FormatJSNumber(t)
	case *dal.Record:
		return "[object Object]"
	}
	return fmt.Sprint(v)
}
