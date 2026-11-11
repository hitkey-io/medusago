package services

import (
	"context"
	"errors"
	"log/slog"

	fulfillmenttypes "github.com/hitkey-io/medusago/packages/core/types/src/fulfillment"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// FulfillmentProviderService mirrors FulfillmentProviderService: the
// registered providers by their registration identifier
// (<identifier>_<id>, registered as fp_<identifier>_<id>).
type FulfillmentProviderService struct {
	providers map[string]fulfillmenttypes.IFulfillmentProvider
	logger    *slog.Logger
}

// NewFulfillmentProviderService registers the configured providers.
func NewFulfillmentProviderService(configured []modulessdk.ProviderInstance, logger *slog.Logger) *FulfillmentProviderService {
	if logger == nil {
		logger = slog.Default()
	}
	s := &FulfillmentProviderService{providers: map[string]fulfillmenttypes.IFulfillmentProvider{}, logger: logger}
	for _, p := range configured {
		if inst, ok := p.Instance.(fulfillmenttypes.IFulfillmentProvider); ok {
			s.providers[p.Identifier+"_"+p.ID] = inst
		}
	}
	return s
}

// retrieveProviderRegistration mirrors retrieveProviderRegistration.
func (s *FulfillmentProviderService) retrieveProviderRegistration(providerID string) (fulfillmenttypes.IFulfillmentProvider, error) {
	if s != nil {
		if p, ok := s.providers[providerID]; ok {
			return p, nil
		}
		s.logger.Error("AwilixResolutionError: Could not resolve 'fp_" + providerID + "'.")
	}
	return nil, errors.New("\nUnable to retrieve the fulfillment provider with id: " + providerID + "\nPlease make sure that the provider is registered in the container and it is configured correctly in your project configuration file.")
}

// GetFulfillmentOptions mirrors getFulfillmentOptions.
func (s *FulfillmentProviderService) GetFulfillmentOptions(ctx context.Context, providerID string) ([]any, error) {
	p, err := s.retrieveProviderRegistration(providerID)
	if err != nil {
		return nil, err
	}
	return p.GetFulfillmentOptions(ctx)
}

// ValidateFulfillmentData mirrors validateFulfillmentData.
func (s *FulfillmentProviderService) ValidateFulfillmentData(ctx context.Context, providerID string, optionData, data, context_ *dal.Record) (any, error) {
	p, err := s.retrieveProviderRegistration(providerID)
	if err != nil {
		return nil, err
	}
	return p.ValidateFulfillmentData(ctx, optionData, data, context_)
}

// ValidateOption mirrors validateOption.
func (s *FulfillmentProviderService) ValidateOption(ctx context.Context, providerID string, data *dal.Record) (bool, error) {
	p, err := s.retrieveProviderRegistration(providerID)
	if err != nil {
		return false, err
	}
	return p.ValidateOption(ctx, data)
}

// CanCalculate mirrors canCalculate.
func (s *FulfillmentProviderService) CanCalculate(ctx context.Context, providerID string, data *dal.Record) (bool, error) {
	p, err := s.retrieveProviderRegistration(providerID)
	if err != nil {
		return false, err
	}
	return p.CanCalculate(ctx, data)
}

// CalculatePrice mirrors calculatePrice.
func (s *FulfillmentProviderService) CalculatePrice(ctx context.Context, providerID string, optionData, data, context_ *dal.Record) (*dal.Record, error) {
	p, err := s.retrieveProviderRegistration(providerID)
	if err != nil {
		return nil, err
	}
	return p.CalculatePrice(ctx, optionData, data, context_)
}

// CreateFulfillment mirrors createFulfillment.
func (s *FulfillmentProviderService) CreateFulfillment(ctx context.Context, providerID string, data *dal.Record, items []any, order, fulfillment *dal.Record) (*dal.Record, error) {
	p, err := s.retrieveProviderRegistration(providerID)
	if err != nil {
		return nil, err
	}
	return p.CreateFulfillment(ctx, data, items, order, fulfillment)
}

// CancelFulfillment mirrors cancelFulfillment.
func (s *FulfillmentProviderService) CancelFulfillment(ctx context.Context, providerID string, data *dal.Record) (any, error) {
	p, err := s.retrieveProviderRegistration(providerID)
	if err != nil {
		return nil, err
	}
	return p.CancelFulfillment(ctx, data)
}

// CreateReturn mirrors createReturn.
func (s *FulfillmentProviderService) CreateReturn(ctx context.Context, providerID string, fulfillment *dal.Record) (*dal.Record, error) {
	p, err := s.retrieveProviderRegistration(providerID)
	if err != nil {
		return nil, err
	}
	return p.CreateReturnFulfillment(ctx, fulfillment)
}
