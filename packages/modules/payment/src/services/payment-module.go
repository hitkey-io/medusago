// Package services mirrors @medusajs/payment/src/services.
package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cockroachdb/apd/v3"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	paymenttypes "github.com/hitkey-io/medusago/packages/core/types/src/payment"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/modules/payment/src/models"
)

// PaymentModuleService mirrors PaymentModuleService.
type PaymentModuleService struct {
	*utilsmodulessdk.MedusaService
	provider *PaymentProviderService
}

// NewPaymentModuleService builds the service.
func NewPaymentModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*PaymentModuleService, error) {
	return &PaymentModuleService{
		MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig),
		provider:      NewPaymentProviderService(opts.Providers, opts.Logger),
	}, nil
}

// PaymentProviderService returns the provider service.
func (s *PaymentModuleService) PaymentProviderService() *PaymentProviderService { return s.provider }

func (s *PaymentModuleService) inTx(ctx context.Context, fn func(tx *PaymentModuleService) error) error {
	return s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		return fn(&PaymentModuleService{MedusaService: tx, provider: s.provider})
	})
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func has(r *dal.Record, k string) bool {
	return r != nil && r.Has(k)
}

func lowerCode(v any) any {
	if s, ok := v.(string); ok && s != "" {
		return strings.ToLower(s)
	}
	return v
}

func merge(rs ...*dal.Record) *dal.Record {
	out := dal.NewRecord()
	for _, r := range rs {
		if r == nil {
			continue
		}
		for _, k := range r.Keys() {
			v, _ := r.Get(k)
			out.Set(k, v)
		}
	}
	return out
}

func asRecord(v any) *dal.Record {
	r, _ := v.(*dal.Record)
	return r
}

// kv builds a record from key/value pairs.
func kv(pairs ...any) *dal.Record {
	r := dal.NewRecord()
	for i := 0; i+1 < len(pairs); i += 2 {
		r.Set(pairs[i].(string), pairs[i+1])
	}
	return r
}

// setDefined sets a key unless the value is undefined (absent).
func setDefined(r *dal.Record, src *dal.Record, k string) {
	if v, ok := src.Get(k); ok {
		r.Set(k, v)
	}
}

// CreatePaymentCollections mirrors createPaymentCollections.
func (s *PaymentModuleService) CreatePaymentCollections(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	in := make([]*dal.Record, len(data))
	for i, d := range data {
		c := merge(d)
		if v := get(d, "currency_code"); v != nil {
			c.Set("currency_code", lowerCode(v))
		}
		in[i] = c
	}
	return s.Internal("PaymentCollection").Create(ctx, in)
}

// CreatePaymentSession mirrors createPaymentSession.
func (s *PaymentModuleService) CreatePaymentSession(ctx context.Context, collectionID string, input *dal.Record) (*dal.Record, error) {
	row := dal.NewRecord()
	row.Set("payment_collection_id", collectionID)
	row.Set("provider_id", get(input, "provider_id"))
	row.Set("amount", get(input, "amount"))
	row.Set("currency_code", lowerCode(get(input, "currency_code")))
	row.Set("context", get(input, "context"))
	// data: undefined keeps the column default ({})
	if v, ok := input.Get("data"); ok && v != nil {
		if r, isRec := v.(*dal.Record); !isRec || r != nil {
			row.Set("data", v)
		}
	}
	row.Set("metadata", get(input, "metadata"))
	created, err := s.Internal("PaymentSession").Create(ctx, []*dal.Record{row})
	if err != nil {
		return nil, err
	}
	session := created[0]
	id := get(session, "id")
	provider := fmt.Sprint(get(input, "provider_id"))
	inputData := asRecord(get(input, "data"))

	fail := func(providerSession *dal.Record, err error) (*dal.Record, error) {
		if providerSession != nil {
			if _, derr := s.provider.DeleteSession(ctx, provider, kv("data", get(input, "data"))); derr != nil {
				return nil, derr
			}
		}
		if _, derr := s.Internal("PaymentSession").Delete(ctx, []any{id}); derr != nil {
			return nil, derr
		}
		return nil, err
	}

	pctx := kv("idempotency_key", id)
	pctx = merge(pctx, asRecord(get(input, "context")))
	pdata := merge(inputData)
	pdata.Set("session_id", id)
	res, err := s.provider.CreateSession(ctx, provider, kv(
		"context", pctx,
		"data", pdata,
		"amount", get(input, "amount"),
		"currency_code", lowerCode(get(input, "currency_code")),
	))
	if err != nil {
		return fail(nil, err)
	}
	status := get(res, "status")
	if status == nil {
		status = paymenttypes.StatusPending
	}
	u := dal.NewRecord()
	u.Set("id", id)
	u.Set("data", merge(inputData, asRecord(get(res, "data"))))
	u.Set("status", status)
	updated, err := s.Internal("PaymentSession").Update(ctx, []any{u})
	if err != nil {
		return fail(res, err)
	}
	rows, _ := updated.([]*dal.Record)
	if len(rows) > 0 {
		return rows[0], nil
	}
	return session, nil
}

// DeletePaymentSession mirrors deletePaymentSession.
func (s *PaymentModuleService) DeletePaymentSession(ctx context.Context, id any) error {
	session, err := s.Internal("PaymentSession").Retrieve(ctx, id, utilsmodulessdk.FindConfig{Select: []string{"id", "data", "provider_id"}})
	if err != nil {
		return err
	}
	if _, err := s.provider.DeleteSession(ctx, fmt.Sprint(get(session, "provider_id")), kv("data", get(session, "data"))); err != nil {
		return err
	}
	_, err = s.Internal("PaymentSession").Delete(ctx, []any{id})
	return err
}

// CreateAccountHolder mirrors createAccountHolder.
func (s *PaymentModuleService) CreateAccountHolder(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	c := asRecord(get(input, "context"))
	if existing, ok := get(c, "account_holder").(*dal.Record); ok && existing != nil {
		return existing, nil
	}
	provider := fmt.Sprint(get(input, "provider_id"))
	customer := asRecord(get(c, "customer"))
	pctx := merge(kv("idempotency_key", get(customer, "id")), c)
	res, err := s.provider.CreateAccountHolder(ctx, provider, kv("context", pctx))
	if err != nil {
		return nil, err
	}
	// isPresent: an empty object means the provider created nothing
	if res == nil || res.Len() == 0 {
		return nil, nil
	}
	row := dal.NewRecord()
	row.Set("external_id", get(res, "id"))
	row.Set("email", get(customer, "email"))
	setDefined(row, res, "data")
	row.Set("provider_id", provider)
	created, err := s.Internal("AccountHolder").Create(ctx, []*dal.Record{row})
	if err != nil {
		return nil, err
	}
	return created[0], nil
}

// UpdateAccountHolder mirrors updateAccountHolder.
func (s *PaymentModuleService) UpdateAccountHolder(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	c := asRecord(get(input, "context"))
	if get(c, "account_holder") == nil {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Missing account holder data while updating account holder.")
	}
	res, err := s.provider.UpdateAccountHolder(ctx, fmt.Sprint(get(input, "provider_id")), kv("context", c))
	if err != nil {
		return nil, err
	}
	u := dal.NewRecord()
	u.Set("id", get(input, "id"))
	if d := get(res, "data"); d != nil {
		u.Set("data", d)
	}
	setDefined(u, input, "metadata")
	updated, err := s.Internal("AccountHolder").Update(ctx, u)
	if err != nil {
		return nil, err
	}
	r, _ := updated.(*dal.Record)
	return r, nil
}

// DeleteAccountHolder mirrors deleteAccountHolder.
func (s *PaymentModuleService) DeleteAccountHolder(ctx context.Context, id any) error {
	holder, err := s.Internal("AccountHolder").Retrieve(ctx, id, utilsmodulessdk.FindConfig{Select: []string{"id", "provider_id", "external_id", "email", "data"}})
	if err != nil {
		return err
	}
	if _, err := s.Internal("AccountHolder").Delete(ctx, []any{id}); err != nil {
		return err
	}
	_, err = s.provider.DeleteAccountHolder(ctx, fmt.Sprint(get(holder, "provider_id")), kv("context", kv("account_holder", holder)))
	return err
}

func paymentMethodsOf(list []*dal.Record, providerID any) []*dal.Record {
	out := make([]*dal.Record, 0, len(list))
	for _, item := range list {
		out = append(out, kv("id", get(item, "id"), "data", get(item, "data"), "provider_id", providerID))
	}
	return out
}

// ListPaymentMethods mirrors listPaymentMethods.
func (s *PaymentModuleService) ListPaymentMethods(ctx context.Context, filters *dal.Record) ([]*dal.Record, error) {
	res, err := s.provider.ListPaymentMethods(ctx, fmt.Sprint(get(filters, "provider_id")), kv("context", get(filters, "context")))
	if err != nil {
		return nil, err
	}
	return paymentMethodsOf(res, get(filters, "provider_id")), nil
}

// ListAndCountPaymentMethods mirrors listAndCountPaymentMethods.
func (s *PaymentModuleService) ListAndCountPaymentMethods(ctx context.Context, filters *dal.Record) ([]*dal.Record, int, error) {
	list, err := s.ListPaymentMethods(ctx, filters)
	return list, len(list), err
}

// CreatePaymentMethods mirrors createPaymentMethods.
func (s *PaymentModuleService) CreatePaymentMethods(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	out := make([]*dal.Record, 0, len(data))
	var errs []string
	for _, item := range data {
		res, err := s.provider.SavePaymentMethod(ctx, fmt.Sprint(get(item, "provider_id")), item)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		out = append(out, kv("id", get(res, "id"), "data", get(res, "data"), "provider_id", get(item, "provider_id")))
	}
	if len(errs) > 0 {
		return nil, aggregateError(errs)
	}
	return out, nil
}

// DeletePaymentMethods mirrors deletePaymentMethods.
func (s *PaymentModuleService) DeletePaymentMethods(ctx context.Context, data []*dal.Record) error {
	var errs []string
	for _, item := range data {
		d := merge(asRecord(get(item, "data")))
		d.Set("id", get(item, "id"))
		if _, err := s.provider.DeletePaymentMethod(ctx, fmt.Sprint(get(item, "provider_id")), kv("context", get(item, "context"), "data", d)); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return aggregateError(errs)
	}
	return nil
}

// aggregateError mirrors promiseAll(..., { aggregateErrors: true }).
func aggregateError(messages []string) error {
	return fmt.Errorf("%s", strings.Join(messages, "\n"))
}

// currencyDigits mirrors Intl.NumberFormat(..., {style: "currency"})
// .resolvedOptions().maximumFractionDigits for the currencies whose minor
// unit is not 2 (ICU data); every other well-formed code has 2.
var currencyDigits = map[string]int32{
	"ADP": 0, "AFN": 0, "ALL": 0, "BIF": 0, "BYR": 0, "CLF": 4, "CLP": 0, "DJF": 0, "ESP": 0, "GNF": 0, "IQD": 0, "IRR": 0,
	"ISK": 0, "ITL": 0, "JPY": 0, "KMF": 0, "KPW": 0, "KRW": 0, "LAK": 0, "LBP": 0, "LUF": 0, "MGA": 0, "MMK": 0, "PYG": 0,
	"RSD": 0, "RWF": 0, "SLL": 0, "SOS": 0, "STD": 0, "SYP": 0, "TMM": 0, "TRL": 0, "UGX": 0, "UYI": 0, "UYW": 4, "VND": 0,
	"VUV": 0, "XAF": 0, "XOF": 0, "XPF": 0, "YER": 0, "ZMK": 0, "ZWD": 0,
	"BHD": 3, "JOD": 3, "KWD": 3, "LYD": 3, "OMR": 3, "TND": 3,
}

// roundToCurrencyPrecision mirrors roundToCurrencyPrecision.
func roundToCurrencyPrecision(amount any, currencyCode string) *totals.BN {
	v := totals.Convert(amount)
	code := strings.ToUpper(currencyCode)
	if len(code) != 3 {
		return v
	}
	digits, ok := currencyDigits[code]
	if !ok {
		digits = 2
	}
	out := new(totals.BN)
	ctx := apd.BaseContext.WithPrecision(100)
	ctx.Rounding = apd.RoundHalfUp
	_, _ = ctx.Quantize(out, v, -digits)
	return out
}

// maybeUpdatePaymentCollection mirrors maybeUpdatePaymentCollection_.
func (s *PaymentModuleService) maybeUpdatePaymentCollection(ctx context.Context, id any) error {
	pc, err := s.Internal("PaymentCollection").Retrieve(ctx, id, utilsmodulessdk.FindConfig{
		Select:    []string{"amount", "raw_amount", "status", "currency_code"},
		Relations: []string{"payment_sessions", "payments.captures", "payments.refunds"},
	})
	if err != nil {
		return err
	}
	sessions := recordList(get(pc, "payment_sessions"))
	authorized, captured, refunded := totals.Convert(0), totals.Convert(0), totals.Convert(0)
	for _, ps := range sessions {
		if get(ps, "status") == "authorized" {
			authorized = totals.Add(authorized, get(ps, "amount"))
		}
	}
	for _, p := range recordList(get(pc, "payments")) {
		for _, c := range recordList(get(p, "captures")) {
			captured = totals.Add(captured, get(c, "amount"))
		}
		for _, r := range recordList(get(p, "refunds")) {
			refunded = totals.Add(refunded, get(r, "amount"))
		}
	}
	status := "awaiting"
	if len(sessions) == 0 {
		status = "not_paid"
	}
	code, _ := get(pc, "currency_code").(string)
	if totals.Gt(authorized, 0) {
		if totals.Gte(roundToCurrencyPrecision(authorized, code), roundToCurrencyPrecision(get(pc, "amount"), code)) {
			status = "authorized"
		} else {
			status = "partially_authorized"
		}
	}
	u := dal.NewRecord()
	u.Set("id", id)
	u.Set("status", status)
	u.Set("authorized_amount", authorized)
	u.Set("captured_amount", captured)
	u.Set("refunded_amount", refunded)
	if totals.Gte(roundToCurrencyPrecision(captured, code), roundToCurrencyPrecision(get(pc, "amount"), code)) {
		u.Set("status", "completed")
		u.Set("completed_at", dal.Now())
	}
	// the collection the caller's entity manager loaded keeps the values
	// read above (upstream returns it with its stale status)
	_, err = s.Internal("PaymentCollection").Update(dal.WithFreshIdentityMap(ctx), []any{u})
	return err
}

func recordList(v any) []*dal.Record {
	switch l := v.(type) {
	case []*dal.Record:
		return l
	case []any:
		var out []*dal.Record
		for _, x := range l {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

// AuthorizePaymentSession mirrors authorizePaymentSession.
func (s *PaymentModuleService) AuthorizePaymentSession(ctx context.Context, id string, context_ *dal.Record) (*dal.Record, error) {
	ctx = dal.WithSharedIdentityMap(ctx)
	session, err := s.Internal("PaymentSession").Retrieve(ctx, id, utilsmodulessdk.FindConfig{
		Select:    []string{"id", "data", "provider_id", "amount", "raw_amount", "currency_code", "authorized_at", "payment_collection_id"},
		Relations: []string{"payment", "payment_collection"},
	})
	if err != nil {
		return nil, err
	}
	if p, ok := get(session, "payment").(*dal.Record); ok && p != nil && get(session, "authorized_at") != nil {
		return p, nil
	}
	provider := fmt.Sprint(get(session, "provider_id"))
	res, err := s.provider.AuthorizePayment(ctx, provider, kv(
		"data", get(session, "data"),
		"context", merge(kv("idempotency_key", get(session, "id")), context_),
	))
	if err != nil {
		return nil, err
	}
	data := get(res, "data")
	status, _ := get(res, "status").(string)
	updateSession := func() error {
		u := dal.NewRecord()
		u.Set("id", id)
		u.Set("status", get(res, "status"))
		setDefined(u, res, "data")
		_, err := s.Internal("PaymentSession").Update(ctx, []any{u})
		return err
	}
	if status == paymenttypes.StatusPendingAuthorization {
		if err := updateSession(); err != nil {
			return nil, err
		}
		return nil, s.maybeUpdatePaymentCollection(ctx, get(session, "payment_collection_id"))
	}
	if status != paymenttypes.StatusAuthorized && status != paymenttypes.StatusCaptured {
		if err := updateSession(); err != nil {
			return nil, err
		}
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, fmt.Sprintf("Session: %s was not authorized with the provider.", id))
	}
	var payment *dal.Record
	err = s.inTx(ctx, func(tx *PaymentModuleService) error {
		var err error
		payment, err = tx.authorizePaymentSession(ctx, session, res, status)
		return err
	})
	if err != nil {
		// payment is never assigned when the transaction fails:
		// { idempotency_key: undefined, ...context }
		cin := kv("data", data, "context", merge(context_))
		if _, cerr := s.provider.CancelPayment(ctx, provider, cin); cerr != nil {
			return nil, cerr
		}
		return nil, err
	}
	if err := s.maybeUpdatePaymentCollection(ctx, get(session, "payment_collection_id")); err != nil {
		return nil, err
	}
	return payment, nil
}

// authorizePaymentSession mirrors authorizePaymentSession_.
func (s *PaymentModuleService) authorizePaymentSession(ctx context.Context, session *dal.Record, res *dal.Record, status string) (*dal.Record, error) {
	isCaptured := false
	if status == paymenttypes.StatusCaptured {
		status = paymenttypes.StatusAuthorized
		isCaptured = true
	}
	id := get(session, "id")
	u := dal.NewRecord()
	u.Set("id", id)
	setDefined(u, res, "data")
	u.Set("status", status)
	if get(session, "authorized_at") == nil {
		u.Set("authorized_at", dal.Now())
	}
	if _, err := s.Internal("PaymentSession").Update(ctx, []any{u}); err != nil {
		return nil, err
	}
	p := dal.NewRecord()
	p.Set("amount", get(session, "amount"))
	p.Set("currency_code", lowerCode(get(session, "currency_code")))
	p.Set("payment_session", id)
	p.Set("payment_collection_id", get(session, "payment_collection_id"))
	p.Set("provider_id", get(session, "provider_id"))
	setDefined(p, res, "data")
	created, err := s.Internal("Payment").Create(ctx, []*dal.Record{p})
	if err != nil {
		return nil, err
	}
	payment := created[0]
	if isCaptured {
		c := dal.NewRecord()
		c.Set("payment_id", get(payment, "id"))
		c.Set("amount", get(session, "amount"))
		c.Set("is_captured", true)
		if _, err := s.capturePayment(ctx, c, true); err != nil {
			return nil, err
		}
	}
	return payment, nil
}

// UpdatePaymentSession mirrors updatePaymentSession: the provider data
// replaces the session data (assigned with mergeObjectProperties) and an
// omitted metadata is assigned as undefined.
func (s *PaymentModuleService) UpdatePaymentSession(ctx context.Context, data *dal.Record) (*dal.Record, error) {
	session, err := s.Internal("PaymentSession").Retrieve(ctx, get(data, "id"), utilsmodulessdk.FindConfig{Select: []string{"id", "status", "data", "provider_id"}})
	if err != nil {
		return nil, err
	}
	pin := dal.NewRecord()
	setDefined(pin, data, "data")
	pin.Set("amount", get(data, "amount"))
	pin.Set("currency_code", lowerCode(get(data, "currency_code")))
	setDefined(pin, data, "context")
	res, err := s.provider.UpdateSession(ctx, fmt.Sprint(get(session, "provider_id")), pin)
	if err != nil {
		return nil, err
	}
	u := dal.NewRecord()
	u.Set("id", get(session, "id"))
	u.Set("amount", get(data, "amount"))
	u.Set("currency_code", lowerCode(get(data, "currency_code")))
	u.Set("data", get(res, "data"))
	status := get(data, "status")
	if status == nil {
		status = get(res, "status")
	}
	if status == nil {
		status = get(session, "status")
	}
	u.Set("status", status)
	u.Set("metadata", get(data, "metadata"))
	updated, err := s.Internal("PaymentSession").Update(ctx, []any{u})
	if err != nil {
		return nil, err
	}
	if list, ok := updated.([]*dal.Record); ok && len(list) > 0 {
		return list[0], nil
	}
	return nil, nil
}

// GetWebhookActionAndData mirrors getWebhookActionAndData.
func (s *PaymentModuleService) GetWebhookActionAndData(ctx context.Context, provider string, payload any) (*dal.Record, error) {
	providerID := provider
	if !strings.HasPrefix(providerID, "pp_") {
		providerID = "pp_" + provider
	}
	p, _ := payload.(*dal.Record)
	if p == nil {
		p = dal.NewRecord()
	}
	return s.provider.GetWebhookActionAndData(ctx, providerID, p)
}

// CompletePaymentCollections mirrors completePaymentCollections: the payment
// collections get completed_at = now.
func (s *PaymentModuleService) CompletePaymentCollections(ctx context.Context, ids []any) ([]*dal.Record, error) {
	now := time.Now()
	in := make([]any, len(ids))
	for i, id := range ids {
		r := dal.NewRecord()
		r.Set("id", id)
		r.Set("completed_at", now)
		in[i] = r
	}
	res, err := s.Internal("PaymentCollection").Update(ctx, in)
	if err != nil {
		return nil, err
	}
	out, _ := res.([]*dal.Record)
	return out, nil
}
