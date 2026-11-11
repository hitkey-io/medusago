package services

import (
	"context"

	analyticstypes "github.com/hitkey-io/medusago/packages/core/types/src/analytics"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// AnalyticsProviderRegistrationPrefix mirrors AnalyticsProviderRegistrationPrefix.
const AnalyticsProviderRegistrationPrefix = "aly_"

// AnalyticsProviderService mirrors AnalyticsProviderService: the module
// uses exactly one provider.
type AnalyticsProviderService struct {
	provider analyticstypes.IAnalyticsProvider
}

// NewAnalyticsProviderService mirrors the constructor.
func NewAnalyticsProviderService(providers []analyticstypes.IAnalyticsProvider) (*AnalyticsProviderService, error) {
	if len(providers) != 1 {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Analytics module should be initialized with exactly one provider")
	}
	return &AnalyticsProviderService{provider: providers[0]}, nil
}

// GetRegistrationIdentifier mirrors getRegistrationIdentifier.
func GetRegistrationIdentifier(identifier, optionName string) string {
	return identifier + "_" + optionName
}

// Track mirrors track.
func (s *AnalyticsProviderService) Track(ctx context.Context, data *dal.Record) error {
	return s.provider.Track(ctx, data)
}

// Identify mirrors identify.
func (s *AnalyticsProviderService) Identify(ctx context.Context, data *dal.Record) error {
	return s.provider.Identify(ctx, data)
}

// Shutdown mirrors shutdown (providers without one are skipped).
func (s *AnalyticsProviderService) Shutdown(ctx context.Context) error {
	if p, ok := s.provider.(analyticstypes.ShutdownProvider); ok {
		return p.Shutdown(ctx)
	}
	return nil
}
