// Package services mirrors @medusajs/payment/src/services.
package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/cockroachdb/apd/v3"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/modules/payment/src/models"
)

// PaymentModuleService mirrors PaymentModuleService.
type PaymentModuleService struct {
	*utilsmodulessdk.MedusaService
}

// NewPaymentModuleService builds the service.
func NewPaymentModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*PaymentModuleService, error) {
	return &PaymentModuleService{MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig)}, nil
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
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

func uuid() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// provider mirrors the payment provider service for the bundled system
// provider (pp_system_default).
type providerResult struct {
	ID     string
	Data   *dal.Record
	Status string
}

func callProvider(providerID, method string) (providerResult, error) {
	if providerID != "pp_system_default" {
		return providerResult{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Could not find a payment provider with id: %s", providerID))
	}
	switch method {
	case "initiatePayment":
		return providerResult{ID: uuid(), Data: dal.NewRecord()}, nil
	case "authorizePayment":
		return providerResult{Data: dal.NewRecord(), Status: "authorized"}, nil
	default:
		return providerResult{Data: dal.NewRecord()}, nil
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
	row.Set("data", get(input, "data"))
	row.Set("metadata", get(input, "metadata"))
	created, err := s.Internal("PaymentSession").Create(ctx, []*dal.Record{row})
	if err != nil {
		return nil, err
	}
	session := created[0]
	id := get(session, "id")
	provider := fmt.Sprint(get(input, "provider_id"))
	res, err := callProvider(provider, "initiatePayment")
	if err != nil {
		_, _ = s.Internal("PaymentSession").Delete(ctx, []any{id})
		return nil, err
	}
	data, _ := get(input, "data").(*dal.Record)
	status := res.Status
	if status == "" {
		status = "pending"
	}
	u := dal.NewRecord()
	u.Set("id", id)
	u.Set("data", merge(data, res.Data))
	u.Set("status", status)
	updated, err := s.Internal("PaymentSession").Update(ctx, []any{u})
	if err != nil {
		_, _ = s.Internal("PaymentSession").Delete(ctx, []any{id})
		return nil, err
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
	if _, err := callProvider(fmt.Sprint(get(session, "provider_id")), "deletePayment"); err != nil {
		return err
	}
	_, err = s.Internal("PaymentSession").Delete(ctx, []any{id})
	return err
}

// CreateAccountHolder mirrors createAccountHolder.
func (s *PaymentModuleService) CreateAccountHolder(ctx context.Context, input *dal.Record) (*dal.Record, error) {
	c, _ := get(input, "context").(*dal.Record)
	if existing, ok := get(c, "account_holder").(*dal.Record); ok && existing != nil {
		return existing, nil
	}
	provider := fmt.Sprint(get(input, "provider_id"))
	if provider != "pp_system_default" {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Could not find a payment provider with id: %s", provider))
	}
	customer, _ := get(c, "customer").(*dal.Record)
	row := dal.NewRecord()
	row.Set("external_id", get(customer, "id"))
	row.Set("email", get(customer, "email"))
	row.Set("provider_id", provider)
	created, err := s.Internal("AccountHolder").Create(ctx, []*dal.Record{row})
	if err != nil {
		return nil, err
	}
	return created[0], nil
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
	_, err = s.Internal("PaymentCollection").Update(ctx, []any{u})
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

// AuthorizePaymentSession mirrors authorizePaymentSession (bundled system
// provider only).
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
	res, err := callProvider(fmt.Sprint(get(session, "provider_id")), "authorizePayment")
	if err != nil {
		return nil, err
	}
	status := res.Status
	if status == "pending_authorization" {
		u := dal.NewRecord()
		u.Set("id", id)
		u.Set("status", status)
		u.Set("data", res.Data)
		if _, err := s.Internal("PaymentSession").Update(ctx, []any{u}); err != nil {
			return nil, err
		}
		return nil, s.maybeUpdatePaymentCollection(ctx, get(session, "payment_collection_id"))
	}
	if status != "authorized" && status != "captured" {
		u := dal.NewRecord()
		u.Set("id", id)
		u.Set("status", status)
		u.Set("data", res.Data)
		if _, err := s.Internal("PaymentSession").Update(ctx, []any{u}); err != nil {
			return nil, err
		}
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, fmt.Sprintf("Session: %s was not authorized with the provider.", id))
	}
	var payment *dal.Record
	err = s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		if status == "captured" {
			return fmt.Errorf("captured payment sessions are not supported yet")
		}
		u := dal.NewRecord()
		u.Set("id", id)
		u.Set("data", res.Data)
		u.Set("status", status)
		if get(session, "authorized_at") == nil {
			u.Set("authorized_at", dal.Now())
		}
		if _, err := tx.Internal("PaymentSession").Update(ctx, []any{u}); err != nil {
			return err
		}
		p := dal.NewRecord()
		p.Set("amount", get(session, "amount"))
		p.Set("currency_code", lowerCode(get(session, "currency_code")))
		p.Set("payment_session", id)
		p.Set("payment_collection_id", get(session, "payment_collection_id"))
		p.Set("provider_id", get(session, "provider_id"))
		p.Set("data", res.Data)
		created, err := tx.Internal("Payment").Create(ctx, []*dal.Record{p})
		if err != nil {
			return err
		}
		payment = created[0]
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.maybeUpdatePaymentCollection(ctx, get(session, "payment_collection_id")); err != nil {
		return nil, err
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
	res, err := callProvider(fmt.Sprint(get(session, "provider_id")), "updatePayment")
	if err != nil {
		return nil, err
	}
	u := dal.NewRecord()
	u.Set("id", get(session, "id"))
	u.Set("amount", get(data, "amount"))
	u.Set("currency_code", lowerCode(get(data, "currency_code")))
	u.Set("data", res.Data)
	status := get(data, "status")
	if status == nil && res.Status != "" {
		status = res.Status
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
