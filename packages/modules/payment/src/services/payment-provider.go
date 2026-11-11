package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	paymenttypes "github.com/hitkey-io/medusago/packages/core/types/src/payment"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/modules/payment/src/providers"
)

// PaymentProviderService mirrors PaymentProviderService: the registered
// providers by their key (pp_<identifier>[_<id>]).
type PaymentProviderService struct {
	providers map[string]paymenttypes.IPaymentProvider
	logger    *slog.Logger
}

// ProviderKey mirrors the registration key `pp_<identifier>[_<id>]`.
func ProviderKey(identifier, id string) string {
	key := "pp_" + identifier
	if id != "" {
		key += "_" + id
	}
	return key
}

// NewPaymentProviderService registers the system provider and the
// configured providers.
func NewPaymentProviderService(configured []modulessdk.ProviderInstance, logger *slog.Logger) *PaymentProviderService {
	if logger == nil {
		logger = slog.Default()
	}
	s := &PaymentProviderService{providers: map[string]paymenttypes.IPaymentProvider{}, logger: logger}
	s.providers[ProviderKey(providers.SystemIdentifier, "default")] = providers.SystemPaymentProvider{}
	for _, p := range configured {
		if inst, ok := p.Instance.(paymenttypes.IPaymentProvider); ok {
			s.providers[ProviderKey(p.Identifier, p.ID)] = inst
		}
	}
	return s
}

// RetrieveProvider mirrors retrieveProvider.
func (s *PaymentProviderService) RetrieveProvider(providerID string) (paymenttypes.IPaymentProvider, error) {
	p, ok := s.providers[providerID]
	if !ok {
		s.logger.Error(fmt.Sprintf("AwilixResolutionError: Could not resolve '%s'.", providerID))
		return nil, errors.New("\nUnable to retrieve the payment provider with id: " + providerID + "\nPlease make sure that the provider is registered in the container and it is configured correctly in your project configuration file.")
	}
	return p, nil
}

func (s *PaymentProviderService) call(providerID string, fn func(p paymenttypes.IPaymentProvider) (*dal.Record, error)) (*dal.Record, error) {
	p, err := s.RetrieveProvider(providerID)
	if err != nil {
		return nil, err
	}
	out, err := fn(p)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = dal.NewRecord()
	}
	return out, nil
}

// CreateSession mirrors createSession.
func (s *PaymentProviderService) CreateSession(ctx context.Context, providerID string, input *dal.Record) (*dal.Record, error) {
	return s.call(providerID, func(p paymenttypes.IPaymentProvider) (*dal.Record, error) { return p.InitiatePayment(ctx, input) })
}

// UpdateSession mirrors updateSession.
func (s *PaymentProviderService) UpdateSession(ctx context.Context, providerID string, input *dal.Record) (*dal.Record, error) {
	return s.call(providerID, func(p paymenttypes.IPaymentProvider) (*dal.Record, error) { return p.UpdatePayment(ctx, input) })
}

// DeleteSession mirrors deleteSession.
func (s *PaymentProviderService) DeleteSession(ctx context.Context, providerID string, input *dal.Record) (*dal.Record, error) {
	return s.call(providerID, func(p paymenttypes.IPaymentProvider) (*dal.Record, error) { return p.DeletePayment(ctx, input) })
}

// AuthorizePayment mirrors authorizePayment.
func (s *PaymentProviderService) AuthorizePayment(ctx context.Context, providerID string, input *dal.Record) (*dal.Record, error) {
	return s.call(providerID, func(p paymenttypes.IPaymentProvider) (*dal.Record, error) { return p.AuthorizePayment(ctx, input) })
}

// GetStatus mirrors getStatus.
func (s *PaymentProviderService) GetStatus(ctx context.Context, providerID string, input *dal.Record) (*dal.Record, error) {
	return s.call(providerID, func(p paymenttypes.IPaymentProvider) (*dal.Record, error) { return p.GetPaymentStatus(ctx, input) })
}

// CapturePayment mirrors capturePayment.
func (s *PaymentProviderService) CapturePayment(ctx context.Context, providerID string, input *dal.Record) (*dal.Record, error) {
	return s.call(providerID, func(p paymenttypes.IPaymentProvider) (*dal.Record, error) { return p.CapturePayment(ctx, input) })
}

// CancelPayment mirrors cancelPayment.
func (s *PaymentProviderService) CancelPayment(ctx context.Context, providerID string, input *dal.Record) (*dal.Record, error) {
	return s.call(providerID, func(p paymenttypes.IPaymentProvider) (*dal.Record, error) { return p.CancelPayment(ctx, input) })
}

// RefundPayment mirrors refundPayment.
func (s *PaymentProviderService) RefundPayment(ctx context.Context, providerID string, input *dal.Record) (*dal.Record, error) {
	return s.call(providerID, func(p paymenttypes.IPaymentProvider) (*dal.Record, error) { return p.RefundPayment(ctx, input) })
}

// RetrieveAccountHolder mirrors retrieveAccountHolder.
func (s *PaymentProviderService) RetrieveAccountHolder(ctx context.Context, providerID string, input *dal.Record) (*dal.Record, error) {
	return s.call(providerID, func(p paymenttypes.IPaymentProvider) (*dal.Record, error) {
		if r, ok := p.(paymenttypes.AccountHolderRetriever); ok {
			return r.RetrieveAccountHolder(ctx, input)
		}
		s.logger.Warn(fmt.Sprintf("Provider %s does not support retrieving account holders", providerID))
		return dal.NewRecord(), nil
	})
}

// CreateAccountHolder mirrors createAccountHolder.
func (s *PaymentProviderService) CreateAccountHolder(ctx context.Context, providerID string, input *dal.Record) (*dal.Record, error) {
	return s.call(providerID, func(p paymenttypes.IPaymentProvider) (*dal.Record, error) {
		if r, ok := p.(paymenttypes.AccountHolderCreator); ok {
			return r.CreateAccountHolder(ctx, input)
		}
		s.logger.Warn(fmt.Sprintf("Provider %s does not support creating account holders", providerID))
		return dal.NewRecord(), nil
	})
}

// UpdateAccountHolder mirrors updateAccountHolder.
func (s *PaymentProviderService) UpdateAccountHolder(ctx context.Context, providerID string, input *dal.Record) (*dal.Record, error) {
	return s.call(providerID, func(p paymenttypes.IPaymentProvider) (*dal.Record, error) {
		if r, ok := p.(paymenttypes.AccountHolderUpdater); ok {
			return r.UpdateAccountHolder(ctx, input)
		}
		s.logger.Warn(fmt.Sprintf("Provider %s does not support updating account holders", providerID))
		return dal.NewRecord(), nil
	})
}

// DeleteAccountHolder mirrors deleteAccountHolder.
func (s *PaymentProviderService) DeleteAccountHolder(ctx context.Context, providerID string, input *dal.Record) (*dal.Record, error) {
	return s.call(providerID, func(p paymenttypes.IPaymentProvider) (*dal.Record, error) {
		if r, ok := p.(paymenttypes.AccountHolderDeleter); ok {
			return r.DeleteAccountHolder(ctx, input)
		}
		s.logger.Warn(fmt.Sprintf("Provider %s does not support deleting account holders", providerID))
		return dal.NewRecord(), nil
	})
}

// ListPaymentMethods mirrors listPaymentMethods.
func (s *PaymentProviderService) ListPaymentMethods(ctx context.Context, providerID string, input *dal.Record) ([]*dal.Record, error) {
	p, err := s.RetrieveProvider(providerID)
	if err != nil {
		return nil, err
	}
	if r, ok := p.(paymenttypes.PaymentMethodLister); ok {
		return r.ListPaymentMethods(ctx, input)
	}
	s.logger.Warn(fmt.Sprintf("Provider %s does not support listing payment methods", providerID))
	return []*dal.Record{}, nil
}

// SavePaymentMethod mirrors savePaymentMethod.
func (s *PaymentProviderService) SavePaymentMethod(ctx context.Context, providerID string, input *dal.Record) (*dal.Record, error) {
	return s.call(providerID, func(p paymenttypes.IPaymentProvider) (*dal.Record, error) {
		if r, ok := p.(paymenttypes.PaymentMethodSaver); ok {
			return r.SavePaymentMethod(ctx, input)
		}
		s.logger.Warn(fmt.Sprintf("Provider %s does not support saving payment methods", providerID))
		return dal.NewRecord(), nil
	})
}

// DeletePaymentMethod mirrors deletePaymentMethod.
func (s *PaymentProviderService) DeletePaymentMethod(ctx context.Context, providerID string, input *dal.Record) (*dal.Record, error) {
	return s.call(providerID, func(p paymenttypes.IPaymentProvider) (*dal.Record, error) {
		if r, ok := p.(paymenttypes.PaymentMethodDeleter); ok {
			return r.DeletePaymentMethod(ctx, input)
		}
		s.logger.Warn(fmt.Sprintf("Provider %s does not support deleting payment methods", providerID))
		return dal.NewRecord(), nil
	})
}

// GetWebhookActionAndData mirrors getWebhookActionAndData.
func (s *PaymentProviderService) GetWebhookActionAndData(ctx context.Context, providerID string, payload *dal.Record) (*dal.Record, error) {
	return s.call(providerID, func(p paymenttypes.IPaymentProvider) (*dal.Record, error) {
		return p.GetWebhookActionAndData(ctx, payload)
	})
}
