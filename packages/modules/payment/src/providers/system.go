// Package providers mirrors @medusajs/payment/src/providers.
package providers

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	paymenttypes "github.com/hitkey-io/medusago/packages/core/types/src/payment"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// SystemPaymentProvider mirrors SystemPaymentProvider.
type SystemPaymentProvider struct{}

// SystemIdentifier mirrors SystemPaymentProvider.identifier.
const SystemIdentifier = "system"

var _ paymenttypes.IPaymentProvider = SystemPaymentProvider{}

func randomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func dataOnly() *dal.Record {
	r := dal.NewRecord()
	r.Set("data", dal.NewRecord())
	return r
}

// InitiatePayment mirrors initiatePayment.
func (SystemPaymentProvider) InitiatePayment(context.Context, *dal.Record) (*dal.Record, error) {
	r := dal.NewRecord()
	r.Set("data", dal.NewRecord())
	r.Set("id", randomUUID())
	return r, nil
}

// GetPaymentStatus mirrors getPaymentStatus.
func (SystemPaymentProvider) GetPaymentStatus(context.Context, *dal.Record) (*dal.Record, error) {
	return nil, errors.New("Method not implemented.")
}

// RetrievePayment mirrors retrievePayment.
func (SystemPaymentProvider) RetrievePayment(context.Context, *dal.Record) (*dal.Record, error) {
	return dal.NewRecord(), nil
}

// AuthorizePayment mirrors authorizePayment.
func (SystemPaymentProvider) AuthorizePayment(context.Context, *dal.Record) (*dal.Record, error) {
	r := dataOnly()
	r.Set("status", paymenttypes.StatusAuthorized)
	return r, nil
}

// UpdatePayment mirrors updatePayment.
func (SystemPaymentProvider) UpdatePayment(context.Context, *dal.Record) (*dal.Record, error) {
	return dataOnly(), nil
}

// DeletePayment mirrors deletePayment.
func (SystemPaymentProvider) DeletePayment(context.Context, *dal.Record) (*dal.Record, error) {
	return dataOnly(), nil
}

// CapturePayment mirrors capturePayment.
func (SystemPaymentProvider) CapturePayment(context.Context, *dal.Record) (*dal.Record, error) {
	return dataOnly(), nil
}

// RetrieveAccountHolder mirrors retrieveAccountHolder.
func (SystemPaymentProvider) RetrieveAccountHolder(_ context.Context, input *dal.Record) (*dal.Record, error) {
	r := dal.NewRecord()
	id, _ := input.Get("id")
	r.Set("id", id)
	return r, nil
}

// CreateAccountHolder mirrors createAccountHolder.
func (SystemPaymentProvider) CreateAccountHolder(_ context.Context, input *dal.Record) (*dal.Record, error) {
	c, _ := get(input, "context").(*dal.Record)
	customer, _ := get(c, "customer").(*dal.Record)
	r := dal.NewRecord()
	r.Set("id", get(customer, "id"))
	return r, nil
}

// DeleteAccountHolder mirrors deleteAccountHolder.
func (SystemPaymentProvider) DeleteAccountHolder(context.Context, *dal.Record) (*dal.Record, error) {
	return dataOnly(), nil
}

// RefundPayment mirrors refundPayment.
func (SystemPaymentProvider) RefundPayment(context.Context, *dal.Record) (*dal.Record, error) {
	return dataOnly(), nil
}

// CancelPayment mirrors cancelPayment.
func (SystemPaymentProvider) CancelPayment(context.Context, *dal.Record) (*dal.Record, error) {
	return dataOnly(), nil
}

// GetWebhookActionAndData mirrors getWebhookActionAndData.
func (SystemPaymentProvider) GetWebhookActionAndData(context.Context, *dal.Record) (*dal.Record, error) {
	r := dal.NewRecord()
	r.Set("action", paymenttypes.ActionNotSupported)
	return r, nil
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}
