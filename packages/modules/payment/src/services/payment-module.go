// Package services mirrors @medusajs/payment/src/services.
package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
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
	row.Set("data", nil)
	row.Set("provider_id", provider)
	created, err := s.Internal("AccountHolder").Create(ctx, []*dal.Record{row})
	if err != nil {
		return nil, err
	}
	return created[0], nil
}
