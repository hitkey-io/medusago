// Package payment mirrors the payment provider types of
// @medusajs/types/src/payment/provider.ts.
//
// Inputs and outputs are JSON-like records, like the plain objects upstream
// passes: a key that is absent is undefined, a key set to nil is null.
package payment

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// IPaymentProvider mirrors IPaymentProvider. Every input carries the keys of
// its upstream type (data, context, amount, currency_code...).
type IPaymentProvider interface {
	// InitiatePayment returns { id, data?, status? }.
	InitiatePayment(ctx context.Context, input *dal.Record) (*dal.Record, error)
	// UpdatePayment returns { data?, status? }.
	UpdatePayment(ctx context.Context, input *dal.Record) (*dal.Record, error)
	// DeletePayment returns { data? }.
	DeletePayment(ctx context.Context, input *dal.Record) (*dal.Record, error)
	// AuthorizePayment returns { data?, status }.
	AuthorizePayment(ctx context.Context, input *dal.Record) (*dal.Record, error)
	// CapturePayment returns { data? }.
	CapturePayment(ctx context.Context, input *dal.Record) (*dal.Record, error)
	// RefundPayment returns { data? }.
	RefundPayment(ctx context.Context, input *dal.Record) (*dal.Record, error)
	// RetrievePayment returns { data? }.
	RetrievePayment(ctx context.Context, input *dal.Record) (*dal.Record, error)
	// CancelPayment returns { data? }.
	CancelPayment(ctx context.Context, input *dal.Record) (*dal.Record, error)
	// GetPaymentStatus returns { data?, status }.
	GetPaymentStatus(ctx context.Context, input *dal.Record) (*dal.Record, error)
	// GetWebhookActionAndData receives the webhook payload
	// { data, rawData, headers } and returns { action, data? }.
	GetWebhookActionAndData(ctx context.Context, payload *dal.Record) (*dal.Record, error)
}

// AccountHolderRetriever is the optional retrieveAccountHolder.
type AccountHolderRetriever interface {
	RetrieveAccountHolder(ctx context.Context, input *dal.Record) (*dal.Record, error)
}

// AccountHolderCreator is the optional createAccountHolder.
type AccountHolderCreator interface {
	CreateAccountHolder(ctx context.Context, input *dal.Record) (*dal.Record, error)
}

// AccountHolderUpdater is the optional updateAccountHolder.
type AccountHolderUpdater interface {
	UpdateAccountHolder(ctx context.Context, input *dal.Record) (*dal.Record, error)
}

// AccountHolderDeleter is the optional deleteAccountHolder.
type AccountHolderDeleter interface {
	DeleteAccountHolder(ctx context.Context, input *dal.Record) (*dal.Record, error)
}

// PaymentMethodLister is the optional listPaymentMethods: it returns the
// provider's payment methods ({ id, data }).
type PaymentMethodLister interface {
	ListPaymentMethods(ctx context.Context, input *dal.Record) ([]*dal.Record, error)
}

// PaymentMethodSaver is the optional savePaymentMethod.
type PaymentMethodSaver interface {
	SavePaymentMethod(ctx context.Context, input *dal.Record) (*dal.Record, error)
}

// PaymentMethodDeleter is the optional deletePaymentMethod.
type PaymentMethodDeleter interface {
	DeletePaymentMethod(ctx context.Context, input *dal.Record) (*dal.Record, error)
}

// PaymentSessionStatus mirrors PaymentSessionStatus.
const (
	StatusAuthorized           = "authorized"
	StatusCaptured             = "captured"
	StatusPending              = "pending"
	StatusRequiresMore         = "requires_more"
	StatusError                = "error"
	StatusCanceled             = "canceled"
	StatusPendingAuthorization = "pending_authorization"
)

// PaymentActions mirrors PaymentActions.
const (
	ActionAuthorized           = "authorized"
	ActionSuccessful           = "captured"
	ActionFailed               = "failed"
	ActionPending              = "pending"
	ActionRequiresMore         = "requires_more"
	ActionCanceled             = "canceled"
	ActionNotSupported         = "not_supported"
	ActionPendingAuthorization = "pending_authorization"
)
