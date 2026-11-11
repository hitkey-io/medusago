// Package services mirrors @medusajs/payment-adyen/src/services.
package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	paymenttypes "github.com/hitkey-io/medusago/packages/core/types/src/payment"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-adyen/src/client"
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-adyen/src/utils"
)

// Identifier mirrors AdyenProviderService.identifier.
const Identifier = "adyen"

// Result codes (PaymentResponse.ResultCodeEnum).
const (
	resultAuthorised          = "Authorised"
	resultRedirectShopper     = "RedirectShopper"
	resultChallengeShopper    = "ChallengeShopper"
	resultIdentifyShopper     = "IdentifyShopper"
	resultCancelled           = "Cancelled"
	resultRefused             = "Refused"
	resultError               = "Error"
	resultPartiallyAuthorised = "PartiallyAuthorised"
)

// Cards and wallets return their result to the storefront's request.
var supportedPaymentMethods = []any{"scheme", "applepay", "googlepay"}

const (
	sessionIDPrefix         = "payses_"
	maxIdempotencyKeyLength = 64
	cardOnFile              = "CardOnFile"
	unscheduledCardOnFile   = "UnscheduledCardOnFile"
	channelWeb              = "Web"
	shopperInteractionEcomm = "Ecommerce"
	shopperInteractionCont  = "ContAuth"
	nativeThreeDSPreferred  = "preferred"
	cardDetailsTypeScheme   = "scheme"
	eventAuthorisation      = "AUTHORISATION"
	eventCancellation       = "CANCELLATION"
	eventCapture            = "CAPTURE"
	eventRefund             = "REFUND"
	eventCaptureFailed      = "CAPTURE_FAILED"
	eventRefundFailed       = "REFUND_FAILED"
)

// applicationInfo identifies payments processed through Medusa.
func applicationInfo() *dal.Record {
	return kv("externalPlatform", kv("name", "Medusa", "integrator", "medusajs"))
}

// AdyenOptions mirrors AdyenOptions.
type AdyenOptions struct {
	APIKey                string
	MerchantAccount       string
	HmacKey               string
	Environment           string
	LiveEndpointURLPrefix string
	Capture               bool
	ReturnURL             *string
}

// AdyenProviderService mirrors AdyenProviderService.
type AdyenProviderService struct {
	options  AdyenOptions
	logger   *slog.Logger
	checkout *client.Client
}

var _ paymenttypes.IPaymentProvider = (*AdyenProviderService)(nil)

func optString(m map[string]any, k string) string {
	s, _ := m[k].(string)
	return s
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
	}
	return true
}

// ValidateOptions mirrors validateOptions.
func ValidateOptions(options map[string]any) error {
	for _, key := range []string{"apiKey", "merchantAccount"} {
		if !truthy(options[key]) {
			return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Required option `%s` is missing in the Adyen payment provider", key))
		}
	}
	if options["environment"] == "LIVE" && !truthy(options["liveEndpointUrlPrefix"]) {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Option `liveEndpointUrlPrefix` is required when `environment` is `LIVE` in the Adyen payment provider")
	}
	return nil
}

// NewAdyenProviderService mirrors the constructor.
func NewAdyenProviderService(options map[string]any, logger *slog.Logger) *AdyenProviderService {
	if logger == nil {
		logger = slog.Default()
	}
	o := AdyenOptions{
		APIKey:                optString(options, "apiKey"),
		MerchantAccount:       optString(options, "merchantAccount"),
		HmacKey:               optString(options, "hmacKey"),
		Environment:           optString(options, "environment"),
		LiveEndpointURLPrefix: optString(options, "liveEndpointUrlPrefix"),
		Capture:               truthy(options["capture"]),
	}
	if s, ok := options["returnUrl"].(string); ok {
		o.ReturnURL = &s
	}
	env := "TEST"
	if o.Environment == "LIVE" {
		env = "LIVE"
	}
	return &AdyenProviderService{options: o, logger: logger, checkout: client.New(client.Config{APIKey: o.APIKey, Environment: env, LiveEndpointURLPrefix: o.LiveEndpointURLPrefix})}
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

func copyRecord(r *dal.Record) *dal.Record {
	out := dal.NewRecord()
	if r == nil {
		return out
	}
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		out.Set(k, v)
	}
	return out
}

// setDefined sets a key unless the value is undefined (nil stands for
// undefined in the request objects built here).
func setDefined(r *dal.Record, k string, v any) {
	if v != nil {
		r.Set(k, v)
	}
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

// hash mirrors sha256(JSON.stringify(value)).
func hash(v any) string {
	raw, _ := dal.MarshalValue(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// toIdempotencyKey mirrors toIdempotencyKey (Adyen allows 64 characters).
func toIdempotencyKey(key string) string {
	if key == "" || len([]rune(key)) <= maxIdempotencyKeyLength {
		return key
	}
	return hash(key)
}

func (s *AdyenProviderService) getShopperReference(c *dal.Record) string {
	return str(get(rec(get(rec(get(c, "account_holder")), "data")), "id"))
}

func orNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// InitiatePayment mirrors initiatePayment.
func (s *AdyenProviderService) InitiatePayment(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	data := rec(get(input, "data"))
	c := rec(get(input, "context"))
	sessionID := get(data, "session_id")
	currency := strings.ToUpper(str(get(input, "currency_code")))
	value := utils.ToMinorUnits(get(input, "amount"), currency)
	countryCode := get(data, "country_code")
	if countryCode == nil {
		countryCode = get(rec(get(rec(get(c, "customer")), "billing_address")), "country_code")
	}
	shopperReference := s.getShopperReference(c)
	offSession := s.getOffSessionPaymentMethod(data, shopperReference)

	req := dal.NewRecord()
	req.Set("merchantAccount", s.options.MerchantAccount)
	req.Set("amount", kv("currency", currency, "value", value))
	if cc, ok := countryCode.(string); ok {
		req.Set("countryCode", strings.ToUpper(cc))
	}
	req.Set("channel", channelWeb)
	setDefined(req, "shopperReference", orNil(shopperReference))
	req.Set("allowedPaymentMethods", supportedPaymentMethods)
	paymentMethods, err := request(s, "initiatePayment", func() (*dal.Record, error) { return s.checkout.PaymentMethods(ctx, req) })
	if err != nil {
		return nil, err
	}
	email := get(rec(get(c, "customer")), "email")
	session := dal.NewRecord()
	session.Set("session_id", sessionID)
	session.Set("amount", value)
	session.Set("currency", currency)
	session.Set("capture", s.options.Capture)
	session.Set("shopper_email", email)
	session.Set("shopper_reference", orNil(shopperReference))
	session.Set("payment_methods", paymentMethods)
	session.Set("psp_reference", nil)
	session.Set("capture_psp_reference", nil)
	session.Set("attempt", float64(0))
	session.Set("off_session_payment_method", orNil(offSession))
	session.Set("result_code", nil)
	session.Set("action", nil)
	return kv("id", sessionID, "status", paymenttypes.StatusPending, "data", session), nil
}

// UpdatePayment mirrors updatePayment.
func (s *AdyenProviderService) UpdatePayment(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	data := rec(get(input, "data"))
	currency := strings.ToUpper(str(get(input, "currency_code")))
	updated := copyRecord(data)
	updated.Set("amount", utils.ToMinorUnits(get(input, "amount"), currency))
	updated.Set("currency", currency)
	client_ := rec(get(data, "client"))
	if truthy(get(client_, "payment_method")) && get(data, "result_code") != resultAuthorised {
		attempt, _ := get(data, "attempt").(float64)
		updated.Set("attempt", attempt+1)
		updated.Set("psp_reference", nil)
		updated.Set("result_code", nil)
		updated.Set("action", nil)
		return kv("status", paymenttypes.StatusPending, "data", updated), nil
	}
	return kv("data", updated), nil
}

// AuthorizePayment mirrors authorizePayment.
func (s *AdyenProviderService) AuthorizePayment(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	data := rec(get(input, "data"))
	session := data
	cl := rec(get(session, "client"))
	if cl == nil {
		cl = dal.NewRecord()
	}
	if get(session, "result_code") == resultAuthorised {
		return kv("status", s.getAuthorizedStatus(session), "data", get(input, "data")), nil
	}
	attempt, _ := get(session, "attempt").(float64)
	sid := str(get(session, "session_id"))
	var response *dal.Record
	var err error
	switch {
	case truthy(get(cl, "details")):
		req := kv("details", get(cl, "details"))
		if pd := get(rec(get(session, "action")), "paymentData"); truthy(pd) {
			req.Set("paymentData", pd)
		}
		key := toIdempotencyKey(fmt.Sprintf("%s-%s", sid, hash(get(cl, "details"))[:16]))
		response, err = request(s, "authorizePayment", func() (*dal.Record, error) {
			return s.checkout.PaymentsDetails(ctx, req, client.RequestOptions{IdempotencyKey: key})
		})
	case truthy(get(cl, "payment_method")):
		req, berr := s.buildPaymentRequest(session, cl)
		if berr != nil {
			return nil, berr
		}
		key := toIdempotencyKey(fmt.Sprintf("%s-%s", sid, jsNum(attempt)))
		response, err = request(s, "authorizePayment", func() (*dal.Record, error) {
			return s.checkout.Payments(ctx, req, client.RequestOptions{IdempotencyKey: key})
		})
	case truthy(get(session, "off_session_payment_method")):
		req := s.buildOffSessionPaymentRequest(session)
		key := toIdempotencyKey(fmt.Sprintf("%s-%s", sid, jsNum(attempt)))
		response, err = request(s, "authorizePayment", func() (*dal.Record, error) {
			return s.checkout.Payments(ctx, req, client.RequestOptions{IdempotencyKey: key})
		})
	default:
		// Nothing was submitted, for example when a webhook authorizes the session.
		return kv("status", s.getStatus(session), "data", get(input, "data")), nil
	}
	if err != nil {
		return nil, err
	}
	resultCode := get(response, "resultCode")
	if resultCode != resultAuthorised {
		reason := ""
		if r := str(get(response, "refusalReason")); r != "" {
			reason = ": " + r
		}
		s.logger.Info(fmt.Sprintf("Adyen payment for session %s returned %s%s", sid, jsString(resultCode), reason))
	}
	var action any
	if a := get(response, "action"); truthy(a) {
		action = a
	}
	isAuthorised := resultCode == resultAuthorised
	updated := copyRecord(session)
	psp := get(response, "pspReference")
	if psp == nil {
		psp = get(session, "psp_reference")
	}
	updated.Set("psp_reference", psp)
	updated.Set("result_code", resultCode)
	updated.Set("action", action)
	if isAuthorised {
		updated.Set("payment_methods", nil)
	} else {
		updated.Set("payment_methods", get(session, "payment_methods"))
	}
	nc := copyRecord(cl)
	nc.Set("payment_method", nil)
	nc.Set("browser_info", nil)
	nc.Set("details", nil)
	updated.Set("client", nc)
	return kv("status", s.getStatus(updated), "data", updated), nil
}

func jsNum(f float64) string {
	return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%f", f), "0"), ".")
}

func jsString(v any) string {
	if v == nil {
		return "undefined"
	}
	return fmt.Sprint(v)
}

// GetPaymentStatus mirrors getPaymentStatus.
func (s *AdyenProviderService) GetPaymentStatus(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	return kv("status", s.getStatus(rec(get(input, "data"))), "data", get(input, "data")), nil
}

// RetrievePayment mirrors retrievePayment: the stored data is the latest
// known state.
func (s *AdyenProviderService) RetrievePayment(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	return kv("data", get(input, "data")), nil
}

func idempotency(input *dal.Record) string {
	return toIdempotencyKey(str(get(rec(get(input, "context")), "idempotency_key")))
}

// CapturePayment mirrors capturePayment.
func (s *AdyenProviderService) CapturePayment(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	session := rec(get(input, "data"))
	if truthy(get(session, "capture")) || truthy(get(session, "capture_psp_reference")) {
		return kv("data", get(input, "data")), nil
	}
	psp, err := s.getPspReference(session)
	if err != nil {
		return nil, err
	}
	req := kv(
		"merchantAccount", s.options.MerchantAccount,
		"amount", kv("currency", get(session, "currency"), "value", get(session, "amount")),
		"reference", get(session, "session_id"),
		"applicationInfo", applicationInfo(),
	)
	response, err := request(s, "capturePayment", func() (*dal.Record, error) {
		return s.checkout.CaptureAuthorisedPayment(ctx, psp, req, client.RequestOptions{IdempotencyKey: idempotency(input)})
	})
	if err != nil {
		return nil, err
	}
	out := copyRecord(session)
	if v, ok := response.Get("pspReference"); ok {
		out.Set("capture_psp_reference", v)
	} else {
		// { ...data, capture_psp_reference: undefined }: the key is dropped
		out.Delete("capture_psp_reference")
	}
	return kv("data", out), nil
}

// RefundPayment mirrors refundPayment.
func (s *AdyenProviderService) RefundPayment(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	session := rec(get(input, "data"))
	psp, err := s.getPspReference(session)
	if err != nil {
		return nil, err
	}
	currency := str(get(session, "currency"))
	req := kv(
		"merchantAccount", s.options.MerchantAccount,
		"amount", kv("currency", get(session, "currency"), "value", utils.ToMinorUnits(get(input, "amount"), currency)),
		"reference", get(session, "session_id"),
		"applicationInfo", applicationInfo(),
	)
	if _, err := request(s, "refundPayment", func() (*dal.Record, error) {
		return s.checkout.RefundCapturedPayment(ctx, psp, req, client.RequestOptions{IdempotencyKey: idempotency(input)})
	}); err != nil {
		return nil, err
	}
	return kv("data", get(input, "data")), nil
}

// CancelPayment mirrors cancelPayment.
func (s *AdyenProviderService) CancelPayment(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	session := rec(get(input, "data"))
	psp := str(get(session, "psp_reference"))
	if psp == "" || get(session, "result_code") != resultAuthorised {
		return kv("data", get(input, "data")), nil
	}
	req := kv(
		"merchantAccount", s.options.MerchantAccount,
		"reference", get(session, "session_id"),
		"applicationInfo", applicationInfo(),
	)
	opts := client.RequestOptions{IdempotencyKey: idempotency(input)}
	// Auto-captured payments can't be canceled, only reversed (refunded).
	if _, err := request(s, "cancelPayment", func() (*dal.Record, error) {
		if truthy(get(session, "capture")) {
			return s.checkout.RefundOrCancelPayment(ctx, psp, req, opts)
		}
		return s.checkout.CancelAuthorisedPaymentByPspReference(ctx, psp, req, opts)
	}); err != nil {
		return nil, err
	}
	return kv("data", get(input, "data")), nil
}

// DeletePayment mirrors deletePayment: an authorization that will never be
// used is released; auto-captured payments aren't refunded.
func (s *AdyenProviderService) DeletePayment(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	session := rec(get(input, "data"))
	if !truthy(get(session, "capture")) {
		return s.CancelPayment(ctx, kv("data", get(input, "data"), "context", get(input, "context")))
	}
	return kv("data", get(input, "data")), nil
}

// CreateAccountHolder mirrors createAccountHolder: the shopperReference is
// the account holder's id.
func (s *AdyenProviderService) CreateAccountHolder(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	c := rec(get(input, "context"))
	if id := get(rec(get(rec(get(c, "account_holder")), "data")), "id"); truthy(id) {
		return kv("id", id), nil
	}
	id := get(rec(get(c, "customer")), "id")
	return kv("id", id, "data", kv("id", id)), nil
}

// RetrieveAccountHolder mirrors retrieveAccountHolder.
func (s *AdyenProviderService) RetrieveAccountHolder(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	id := get(input, "id")
	return kv("id", id, "data", kv("id", id)), nil
}

// UpdateAccountHolder mirrors updateAccountHolder.
func (s *AdyenProviderService) UpdateAccountHolder(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	return dal.NewRecord(), nil
}

// DeleteAccountHolder mirrors deleteAccountHolder: the shopper's saved cards
// are deleted too.
func (s *AdyenProviderService) DeleteAccountHolder(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	ref := s.getShopperReference(rec(get(input, "context")))
	if ref == "" {
		return dal.NewRecord(), nil
	}
	methods, err := s.listStoredPaymentMethods(ctx, ref)
	if err != nil {
		return nil, err
	}
	failed := 0
	for _, m := range methods {
		if err := s.deleteStoredPaymentMethod(ctx, str(get(m, "id")), ref); err != nil {
			failed++
		}
	}
	if failed > 0 {
		s.logger.Error(fmt.Sprintf("Adyen failed to delete %d of %d saved cards of shopper %s", failed, len(methods), ref))
	}
	return dal.NewRecord(), nil
}

// ListPaymentMethods mirrors listPaymentMethods.
func (s *AdyenProviderService) ListPaymentMethods(ctx context.Context, input *dal.Record) ([]*dal.Record, error) {
	ref := s.getShopperReference(rec(get(input, "context")))
	if ref == "" {
		return []*dal.Record{}, nil
	}
	methods, err := s.listStoredPaymentMethods(ctx, ref)
	if err != nil {
		return nil, err
	}
	out := []*dal.Record{}
	for _, m := range methods {
		out = append(out, kv("id", get(m, "id"), "data", m))
	}
	return out, nil
}

// SavePaymentMethod mirrors savePaymentMethod.
func (s *AdyenProviderService) SavePaymentMethod(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	c := rec(get(input, "context"))
	ref, err := s.requireShopperReference(c)
	if err != nil {
		return nil, err
	}
	data := rec(get(input, "data"))
	model := cardOnFile
	if get(data, "usage") == "off_session" {
		model = unscheduledCardOnFile
	}
	req := dal.NewRecord()
	req.Set("merchantAccount", s.options.MerchantAccount)
	req.Set("shopperReference", ref)
	setDefined(req, "paymentMethod", get(data, "payment_method"))
	req.Set("recurringProcessingModel", model)
	setDefined(req, "shopperEmail", get(rec(get(c, "customer")), "email"))
	pm, err := request(s, "savePaymentMethod", func() (*dal.Record, error) {
		return s.checkout.StoredPaymentMethods(ctx, req, client.RequestOptions{IdempotencyKey: idempotency(input)})
	})
	if err != nil {
		return nil, err
	}
	return kv("id", get(pm, "id"), "data", pm), nil
}

// DeletePaymentMethod mirrors deletePaymentMethod.
func (s *AdyenProviderService) DeletePaymentMethod(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	ref, err := s.requireShopperReference(rec(get(input, "context")))
	if err != nil {
		return nil, err
	}
	if err := s.deleteStoredPaymentMethod(ctx, str(get(rec(get(input, "data")), "id")), ref); err != nil {
		return nil, err
	}
	return dal.NewRecord(), nil
}

func (s *AdyenProviderService) listStoredPaymentMethods(ctx context.Context, ref string) ([]*dal.Record, error) {
	res, err := request(s, "listPaymentMethods", func() (*dal.Record, error) {
		return s.checkout.GetTokensForStoredPaymentDetails(ctx, ref, s.options.MerchantAccount)
	})
	if err != nil {
		return nil, err
	}
	var out []*dal.Record
	if l, ok := get(res, "storedPaymentMethods").([]any); ok {
		for _, x := range l {
			out = append(out, rec(x))
		}
	}
	return out, nil
}

func (s *AdyenProviderService) deleteStoredPaymentMethod(ctx context.Context, id, ref string) error {
	_, err := request(s, "deletePaymentMethod", func() (*dal.Record, error) {
		return nil, s.checkout.DeleteTokenForStoredPaymentDetails(ctx, id, ref, s.options.MerchantAccount)
	})
	return err
}

// GetWebhookActionAndData mirrors getWebhookActionAndData.
func (s *AdyenProviderService) GetWebhookActionAndData(ctx context.Context, payload *dal.Record) (*dal.Record, error) {
	notSupported := kv("action", paymenttypes.ActionNotSupported)
	data := rec(get(payload, "data"))
	var items []*dal.Record
	if l, ok := get(data, "notificationItems").([]any); ok {
		for _, x := range l {
			item := rec(get(rec(x), "NotificationRequestItem"))
			if s.isValidNotification(item) {
				items = append(items, item)
			}
		}
	}
	if len(items) > 1 {
		s.logger.Warn(fmt.Sprintf("Adyen webhook contained %d notifications, only the first is processed", len(items)))
	}
	if len(items) == 0 {
		return notSupported, nil
	}
	item := items[0]
	succeeded := get(item, "success") == "true"
	amount := rec(get(item, "amount"))
	value := get(amount, "value")
	if value == nil {
		value = float64(0)
	}
	currency := str(get(amount, "currency"))
	var am any = float64(0)
	if currency != "" {
		am = utils.FromMinorUnits(value, currency)
	}
	result := kv("session_id", get(item, "merchantReference"), "amount", am)
	eventCode := get(item, "eventCode")
	switch eventCode {
	case eventAuthorisation:
		if !succeeded {
			return kv("action", paymenttypes.ActionFailed, "data", result), nil
		}
		action := paymenttypes.ActionAuthorized
		if s.options.Capture {
			action = paymenttypes.ActionSuccessful
		}
		return kv("action", action, "data", result), nil
	case eventCancellation:
		if succeeded {
			return kv("action", paymenttypes.ActionCanceled, "data", result), nil
		}
		return notSupported, nil
	case eventCapture, eventRefund:
		if !succeeded {
			s.logger.Error(fmt.Sprintf("Adyen %v for session %v failed: %s", eventCode, jsString(get(item, "merchantReference")), jsString(get(item, "reason"))))
		}
		return notSupported, nil
	case eventCaptureFailed, eventRefundFailed:
		s.logger.Error(fmt.Sprintf("Adyen %v for session %v: %s", eventCode, jsString(get(item, "merchantReference")), jsString(get(item, "reason"))))
		return notSupported, nil
	}
	return notSupported, nil
}

// isValidNotification mirrors isValidNotification.
func (s *AdyenProviderService) isValidNotification(item *dal.Record) bool {
	if item == nil {
		return false
	}
	if s.options.HmacKey == "" {
		s.logger.Warn("Adyen webhook rejected because the `hmacKey` option isn't set")
		return false
	}
	valid, err := client.ValidateHMAC(item, s.options.HmacKey)
	if err != nil {
		valid = false
	}
	if !valid {
		s.logger.Warn(fmt.Sprintf("Adyen webhook for %s rejected because of an invalid HMAC signature", jsString(get(item, "pspReference"))))
		return false
	}
	return get(item, "merchantAccountCode") == s.options.MerchantAccount &&
		strings.HasPrefix(str(get(item, "merchantReference")), sessionIDPrefix)
}

func amountOf(session *dal.Record) *dal.Record {
	return kv("currency", get(session, "currency"), "value", get(session, "amount"))
}

// captureFields mirrors `...(session.capture ? { captureDelayHours: 0 } :
// { additionalData: { manualCapture: "true" } })`.
func captureFields(r *dal.Record, session *dal.Record) {
	if truthy(get(session, "capture")) {
		r.Set("captureDelayHours", float64(0))
	} else {
		r.Set("additionalData", kv("manualCapture", "true"))
	}
}

// buildPaymentRequest mirrors buildPaymentRequest.
func (s *AdyenProviderService) buildPaymentRequest(session, cl *dal.Record) (*dal.Record, error) {
	pm := rec(get(cl, "payment_method"))
	if t := str(get(pm, "type")); t != "" {
		supported := false
		for _, x := range supportedPaymentMethods {
			if x == t {
				supported = true
			}
		}
		if !supported {
			return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Adyen payment method %s isn't supported", t))
		}
	}
	returnURL := get(cl, "return_url")
	if returnURL == nil && s.options.ReturnURL != nil {
		returnURL = *s.options.ReturnURL
	}
	if !truthy(returnURL) {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Adyen requires a return URL. Set the `returnUrl` option or pass `return_url` in the payment session's client data.")
	}
	shopperReference := get(session, "shopper_reference")
	isStored := truthy(get(pm, "storedPaymentMethodId"))
	storePaymentMethod := truthy(shopperReference) && truthy(get(cl, "store_payment_method"))
	r := dal.NewRecord()
	r.Set("merchantAccount", s.options.MerchantAccount)
	r.Set("amount", amountOf(session))
	r.Set("reference", get(session, "session_id"))
	setDefined(r, "shopperReference", shopperReference)
	if storePaymentMethod {
		r.Set("storePaymentMethod", true)
	}
	if storePaymentMethod || isStored {
		r.Set("recurringProcessingModel", cardOnFile)
	}
	r.Set("paymentMethod", get(cl, "payment_method"))
	setDefined(r, "browserInfo", get(cl, "browser_info"))
	setDefined(r, "origin", get(cl, "origin"))
	r.Set("returnUrl", returnURL)
	setDefined(r, "shopperEmail", get(session, "shopper_email"))
	r.Set("channel", channelWeb)
	r.Set("shopperInteraction", shopperInteractionEcomm)
	r.Set("authenticationData", kv("threeDSRequestData", kv("nativeThreeDS", nativeThreeDSPreferred)))
	captureFields(r, session)
	r.Set("applicationInfo", applicationInfo())
	return r, nil
}

// getOffSessionPaymentMethod mirrors getOffSessionPaymentMethod.
func (s *AdyenProviderService) getOffSessionPaymentMethod(data *dal.Record, shopperReference string) string {
	if get(data, "off_session") != true || shopperReference == "" {
		return ""
	}
	pm, ok := get(data, "payment_method").(string)
	if !ok {
		return ""
	}
	return pm
}

// buildOffSessionPaymentRequest mirrors buildOffSessionPaymentRequest.
func (s *AdyenProviderService) buildOffSessionPaymentRequest(session *dal.Record) *dal.Record {
	r := dal.NewRecord()
	r.Set("merchantAccount", s.options.MerchantAccount)
	r.Set("amount", amountOf(session))
	r.Set("reference", get(session, "session_id"))
	setDefined(r, "shopperReference", get(session, "shopper_reference"))
	r.Set("paymentMethod", kv("type", cardDetailsTypeScheme, "storedPaymentMethodId", get(session, "off_session_payment_method")))
	setDefined(r, "shopperEmail", get(session, "shopper_email"))
	r.Set("shopperInteraction", shopperInteractionCont)
	r.Set("recurringProcessingModel", unscheduledCardOnFile)
	captureFields(r, session)
	r.Set("applicationInfo", applicationInfo())
	return r
}

// getStatus mirrors getStatus.
func (s *AdyenProviderService) getStatus(session *dal.Record) string {
	switch get(session, "result_code") {
	case resultAuthorised:
		return s.getAuthorizedStatus(session)
	case resultRedirectShopper, resultChallengeShopper, resultIdentifyShopper:
		return paymenttypes.StatusRequiresMore
	case resultCancelled:
		return paymenttypes.StatusCanceled
	case resultRefused, resultError, resultPartiallyAuthorised:
		return paymenttypes.StatusError
	}
	return paymenttypes.StatusPending
}

func (s *AdyenProviderService) getAuthorizedStatus(session *dal.Record) string {
	if truthy(get(session, "capture")) {
		return paymenttypes.StatusCaptured
	}
	return paymenttypes.StatusAuthorized
}

func (s *AdyenProviderService) requireShopperReference(c *dal.Record) (string, error) {
	ref := s.getShopperReference(c)
	if ref == "" {
		return "", common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Adyen saved payment methods require an account holder")
	}
	return ref, nil
}

func (s *AdyenProviderService) getPspReference(session *dal.Record) (string, error) {
	psp := str(get(session, "psp_reference"))
	if psp == "" {
		return "", common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Adyen payment for session %s has no PSP reference", jsString(get(session, "session_id"))))
	}
	return psp, nil
}

// request mirrors request: Adyen's error details are logged, not returned.
func request(s *AdyenProviderService, operation string, fn func() (*dal.Record, error)) (*dal.Record, error) {
	out, err := fn()
	if err == nil {
		return out, nil
	}
	statusCode := 0
	var parts []string
	var he *client.HTTPClientException
	if errors.As(err, &he) {
		statusCode = he.StatusCode
		if statusCode != 0 {
			parts = append(parts, fmt.Sprint(statusCode))
		}
		if he.ErrorCode != "" {
			parts = append(parts, he.ErrorCode)
		}
		if he.Message != "" {
			parts = append(parts, he.Message)
		}
		if he.ResponseBody != "" {
			parts = append(parts, he.ResponseBody)
		}
	} else if err.Error() != "" {
		parts = append(parts, err.Error())
	}
	s.logger.Error(fmt.Sprintf("Adyen %s failed: %s", operation, strings.Join(parts, " | ")))
	isValidation := statusCode >= 400 && statusCode < 500 && statusCode != 401 && statusCode != 403
	t := common.MedusaErrorTypes.UNEXPECTED_STATE
	if isValidation {
		t = common.MedusaErrorTypes.INVALID_DATA
	}
	return nil, common.NewMedusaError(t, fmt.Sprintf("An error occurred in %s with Adyen", operation))
}
